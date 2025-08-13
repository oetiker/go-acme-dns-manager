package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/common"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// Config holds application configuration
type Config struct {
	ConfigPath          string
	AutoMode            bool
	QuietMode           bool
	PrintConfigTemplate bool
	DebugMode           bool
	LogLevel            string
	LogFormat           string
	ShowVersion         bool
	Version             string
}

// Application represents the main application with dependency injection
type Application struct {
	config     *Config
	logger     common.LoggerInterface
	flags      *Flags
	cancelFunc context.CancelFunc
	done       chan struct{}
	shutdownOnce sync.Once
}

// Flags encapsulates command line flag parsing
type Flags struct {
	configPath          *string
	autoMode            *bool
	quietMode           *bool
	printConfigTemplate *bool
	debugMode           *bool
	logLevel            *string
	logFormat           *string
	showVersion         *bool
}

// NewApplication creates a new application instance
func NewApplication(version string) *Application {
	return &Application{
		config: &Config{Version: version},
		flags:  &Flags{},
		done:   make(chan struct{}),
	}
}

// SetupFlags configures command line flags
func (app *Application) SetupFlags() {
	app.flags.configPath = flag.String("config", "config.yaml", "Path to the configuration file")
	app.flags.autoMode = flag.Bool("auto", false, "Enable automatic mode using 'auto_domains' config section (handles init and renew)")
	app.flags.quietMode = flag.Bool("quiet", false, "Reduce output in auto mode (useful for cron jobs)")
	app.flags.printConfigTemplate = flag.Bool("print-config-template", false, "Print a default configuration template to stdout and exit")
	app.flags.debugMode = flag.Bool("debug", false, "Enable debug logging")
	app.flags.logLevel = flag.String("log-level", "", "Set logging level (debug|info|warn|error), overrides -debug flag if specified")
	app.flags.logFormat = flag.String("log-format", "", "Set logging format (go|emoji|color|ascii), overrides -no-color and -no-emoji flags")
	app.flags.showVersion = flag.Bool("version", false, "Show version information and exit")

	flag.Usage = app.printUsage
}

// ParseFlags parses command line flags and populates config
func (app *Application) ParseFlags() {
	flag.Parse()

	app.config.ConfigPath = *app.flags.configPath
	app.config.AutoMode = *app.flags.autoMode
	app.config.QuietMode = *app.flags.quietMode
	app.config.PrintConfigTemplate = *app.flags.printConfigTemplate
	app.config.DebugMode = *app.flags.debugMode
	app.config.LogLevel = *app.flags.logLevel
	app.config.LogFormat = *app.flags.logFormat
	app.config.ShowVersion = *app.flags.showVersion
}

// printUsage prints application usage information
func (app *Application) printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags] [cert-name@domain1,domain2.../key_type=TYPE... [cert-name2@domain3...]]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  Manages ACME certificates using acme-dns.\n\n")
	fmt.Fprintf(os.Stderr, "Modes:\n")
	fmt.Fprintf(os.Stderr, "  Manual Mode: Provide one or more certificate requests as arguments.\n")
	fmt.Fprintf(os.Stderr, "             Example: %s -config my.yaml cert1@example.com,www.example.com/key_type=ec384 cert2@service.example.com\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  Automatic Mode: Use the -auto flag (no certificate arguments allowed).\n")
	fmt.Fprintf(os.Stderr, "                  Processes certificates defined in the 'auto_domains' section of the config file (handles init and renew).\n")
	fmt.Fprintf(os.Stderr, "             Example: %s -config my.yaml -auto\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  Key Types: rsa2048, rsa3072, rsa4096, ec256, ec384\n\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

// HandleVersionFlag handles the version display flag
func (app *Application) HandleVersionFlag() bool {
	if app.config.ShowVersion {
		fmt.Printf("go-acme-dns-manager %s\n", app.config.Version)
		fmt.Printf("Build date: %s\n", time.Now().Format("2006-01-02"))
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return true
	}
	return false
}

// SetupLogger configures the application logger
func (app *Application) SetupLogger() error {
	loggerLevel := manager.LogLevelInfo // Default log level
	var loggerFormat manager.LogFormat  // Will be initialized later

	// Parse log level flag if specified
	if app.config.LogLevel != "" {
		switch strings.ToLower(app.config.LogLevel) {
		case "debug":
			loggerLevel = manager.LogLevelDebug
		case "info":
			loggerLevel = manager.LogLevelInfo
		case "warn", "warning":
			loggerLevel = manager.LogLevelWarn
		case "error":
			loggerLevel = manager.LogLevelError
		default:
			fmt.Fprintf(os.Stderr, "Invalid log level: %s. Using default (info).\n", app.config.LogLevel)
		}
	} else {
		// Use the legacy flags if log-level is not specified
		if app.config.QuietMode && app.config.AutoMode {
			loggerLevel = manager.LogLevelQuiet
		} else if app.config.DebugMode {
			loggerLevel = manager.LogLevelDebug
		}
	}

	// Parse log format flag if specified
	if app.config.LogFormat != "" {
		switch strings.ToLower(app.config.LogFormat) {
		case "go":
			loggerFormat = manager.LogFormatGo
		case "emoji":
			loggerFormat = manager.LogFormatEmoji
		case "color":
			loggerFormat = manager.LogFormatColor
		case "ascii":
			loggerFormat = manager.LogFormatASCII
		default:
			fmt.Fprintf(os.Stderr, "Invalid log format: %s. Using default.\n", app.config.LogFormat)
			loggerFormat = manager.LogFormatDefault
		}
	} else {
		// Set format based on legacy flags
		loggerFormat = manager.LogFormatDefault
	}

	// Set up the logger
	manager.SetupDefaultLogger(loggerLevel, loggerFormat)
	app.logger = manager.GetDefaultLogger()

	return nil
}

// HandleConfigTemplate handles the config template printing
func (app *Application) HandleConfigTemplate() bool {
	if app.config.PrintConfigTemplate {
		fmt.Println("# Default configuration template:")
		err := manager.GenerateDefaultConfig(os.Stdout)
		if err != nil {
			app.logger.Errorf("Error printing config template: %v", err)
			return true
		}
		return true
	}
	return false
}

// LoadConfiguration loads and validates the configuration file
func (app *Application) LoadConfiguration() (*manager.Config, error) {
	return app.LoadConfigurationWithContext(context.Background())
}

// LoadConfigurationWithContext loads and validates the configuration file with context support
func (app *Application) LoadConfigurationWithContext(ctx context.Context) (*manager.Config, error) {
	// Check for cancellation before starting
	if common.IsContextCanceled(ctx) {
		return nil, common.GetContextError(ctx, "load configuration")
	}

	absConfigPath, err := filepath.Abs(app.config.ConfigPath)
	if err != nil {
		return nil, common.WrapError(err, common.ErrorTypeConfig, "resolve config path",
			"Failed to resolve absolute path for configuration file").
			AddContext("config_path", app.config.ConfigPath).
			AddContext("request_id", common.GetRequestID(ctx)).
			AddSuggestion("Check that the config path is valid and accessible")
	}
	app.config.ConfigPath = absConfigPath

	// Check for cancellation after path resolution
	if common.IsContextCanceled(ctx) {
		return nil, common.GetContextError(ctx, "load configuration")
	}

	// Check if config file exists
	if _, err := os.Stat(app.config.ConfigPath); os.IsNotExist(err) {
		return nil, common.NewConfigError("locate config file",
			"Configuration file not found").
			AddContext("config_path", app.config.ConfigPath).
			AddContext("request_id", common.GetRequestID(ctx)).
			AddSuggestion("Use -print-config-template to generate a template").
			AddSuggestion("Ensure the file path is correct")
	} else if err != nil {
		return nil, common.WrapError(err, common.ErrorTypeStorage, "access config file",
			"Failed to access configuration file").
			AddContext("config_path", app.config.ConfigPath).
			AddContext("request_id", common.GetRequestID(ctx))
	}

	// Check for cancellation before file operations
	if common.IsContextCanceled(ctx) {
		return nil, common.GetContextError(ctx, "load configuration")
	}

	// Load configuration
	app.logger.Infof("Loading configuration from %s... (request: %s)",
		app.config.ConfigPath, common.GetRequestID(ctx))

	cfg, err := manager.LoadConfig(app.config.ConfigPath)
	if err != nil {
		// Check for placeholder email
		contentBytes, readErr := os.ReadFile(app.config.ConfigPath)
		if readErr == nil {
			content := string(contentBytes)
			if strings.Contains(content, "your-email@example.com") {
				return nil, common.NewConfigError("validate config content",
					"Configuration file contains placeholder email address").
					AddContext("config_path", app.config.ConfigPath).
					AddContext("request_id", common.GetRequestID(ctx)).
					AddSuggestion("Replace 'your-email@example.com' with your actual email address").
					AddSuggestion("Edit the configuration file before running the application")
			}
		}
		return nil, common.WrapError(err, common.ErrorTypeConfig, "parse config file",
			"Failed to parse configuration file").
			AddContext("config_path", app.config.ConfigPath).
			AddContext("request_id", common.GetRequestID(ctx))
	}

	// Final cancellation check
	if common.IsContextCanceled(ctx) {
		return nil, common.GetContextError(ctx, "load configuration")
	}

	app.logger.Infof("Configuration loaded successfully. (request: %s)", common.GetRequestID(ctx))
	return cfg, nil
}

// ValidateMode validates the operation mode (manual vs auto)
func (app *Application) ValidateMode() error {
	return app.ValidateModeWithArgs(flag.Args())
}

// ValidateModeWithArgs validates the operation mode with explicit arguments (testable version)
func (app *Application) ValidateModeWithArgs(args []string) error {
	isManualMode := len(args) > 0
	isAutoMode := app.config.AutoMode

	if isManualMode && isAutoMode {
		return common.NewValidationError("validate operation mode",
			"Cannot use automatic mode and manual certificate arguments simultaneously").
			AddContext("auto_mode", isAutoMode).
			AddContext("manual_args_count", len(args)).
			AddSuggestion("Remove the -auto flag to use manual mode").
			AddSuggestion("Remove certificate arguments to use automatic mode")
	}

	if !isManualMode && !isAutoMode {
		return common.NewValidationError("validate operation mode",
			"No operation specified").
			AddContext("auto_mode", isAutoMode).
			AddContext("manual_args_count", len(args)).
			AddSuggestion("Use -auto flag for automatic mode").
			AddSuggestion("Provide certificate arguments for manual mode").
			AddSuggestion("Example: cert-name@domain1,domain2 or use -auto")
	}

	return nil
}

// setupGracefulShutdown sets up signal handling for graceful shutdown
func (app *Application) setupGracefulShutdown(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	app.cancelFunc = cancel

	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine to handle signals
	go func() {
		select {
		case sig := <-sigChan:
			if app.logger != nil {
				app.logger.Infof("Received signal %v, initiating graceful shutdown...", sig)
			}
			app.Shutdown()
		case <-ctx.Done():
			// Context was canceled, cleanup
			app.Shutdown()
		}
	}()

	return ctx
}

// Shutdown gracefully shuts down the application
// This method is safe to call multiple times
func (app *Application) Shutdown() {
	app.shutdownOnce.Do(func() {
		if app.logger != nil {
			app.logger.Info("Shutting down application...")
		}

		if app.cancelFunc != nil {
			app.cancelFunc()
		}

		close(app.done)
	})
}

// WaitForShutdown waits for the application to shutdown
func (app *Application) WaitForShutdown() {
	<-app.done
}

// Run executes the main application logic with context support
func (app *Application) Run(ctx context.Context) error {
	// Setup graceful shutdown handling
	ctx = app.setupGracefulShutdown(ctx)

	// Add request tracing
	ctx = common.WithRequestID(ctx)
	ctx = common.WithOperation(ctx, "application_startup")

	// Handle early exit flags
	if app.HandleVersionFlag() {
		app.Shutdown()
		return nil
	}

	if app.HandleConfigTemplate() {
		app.Shutdown()
		return nil
	}

	// Setup logger first so we can use it for version output
	if err := app.SetupLogger(); err != nil {
		return fmt.Errorf("setting up logger: %w", err)
	}

	// Display version at info level (hidden in quiet mode)
	app.logger.Infof("go-acme-dns-manager %s", app.config.Version)

	app.logger.Debugf("Starting application with request ID: %s", common.GetRequestID(ctx))

	// Load configuration with timeout
	configCtx, configCancel := common.WithOperationTimeout(ctx)
	defer configCancel()

	_, err := app.LoadConfigurationWithContext(configCtx)
	if err != nil {
		// Check if it was a context error
		if ctxErr := common.GetContextError(configCtx, "load configuration"); ctxErr != nil {
			return ctxErr
		}
		return err
	}

	// Validate mode
	if err := app.ValidateMode(); err != nil {
		return err
	}

	// Continue with certificate processing...
	// Load manager config and create certificate manager
	managerConfig, err := app.LoadManagerConfig()
	if err != nil {
		return fmt.Errorf("loading manager config: %w", err)
	}

	certManager, err := NewCertificateManager(managerConfig, app.logger)
	if err != nil {
		return fmt.Errorf("creating certificate manager: %w", err)
	}

	// Process certificates based on mode
	var processingErr error
	if app.config.AutoMode {
		app.logger.Info("Starting automatic certificate processing...")
		processingErr = certManager.ProcessAutoMode(ctx)
	} else {
		app.logger.Info("Starting manual certificate processing...")
		args := flag.Args()
		if len(args) == 0 {
			return fmt.Errorf("no certificate requests provided in manual mode")
		}
		processingErr = certManager.ProcessManualMode(ctx, args)
	}

	// Handle processing result
	if processingErr != nil {
		// Check if this is just DNS setup needed (not really an error)
		if errors.Is(processingErr, manager.ErrDNSSetupNeeded) {
			// DNS instructions were already shown, exit cleanly
			// Use Warn level so it shows even in quiet mode
			app.logger.Warn("Please configure the DNS records as shown above and run the command again.")
			app.Shutdown() // Signal that we're done so WaitForShutdown doesn't hang
			return nil
		}
		mode := "auto"
		if !app.config.AutoMode {
			mode = "manual"
		}
		return fmt.Errorf("processing certificates in %s mode: %w", mode, processingErr)
	}

	app.logger.Info("Certificate processing completed successfully")

	// Check if we were asked to shutdown during startup
	if common.IsContextCanceled(ctx) {
		return common.GetContextError(ctx, "application startup")
	}

	// Shutdown normally after completing work
	app.Shutdown()
	return nil
}

// LoadManagerConfig loads the manager configuration from the parsed config
func (app *Application) LoadManagerConfig() (*manager.Config, error) {
	app.logger.Debug("Loading manager configuration...")

	// Load the configuration file
	cfg, err := manager.LoadConfig(app.config.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("loading config file: %w", err)
	}

	app.logger.Debug("Manager configuration loaded successfully")
	return cfg, nil
}
