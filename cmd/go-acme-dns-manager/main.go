package main

import (
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath" // For comparing domain lists
	"runtime"

	// "sort" // Removed unused import
	"strings"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// Define a struct to hold parsed certificate requests
type certRequest struct {
	Name    string
	Domains []string
	KeyType string
}

// parseCertArg parses certificate arguments in the format cert-name@domain1,domain2/key_type=ec384
// This extracts certificate name, domains list, and optional key type parameter
func parseCertArg(arg string) (string, []string, string, error) {
	// Check for key_type parameter
	keyType := ""
	domainPart := arg

	// Special case: Check for slash in the cert name part, which is an invalid format
	// Must handle this before processing parameters
	atIndex := strings.Index(arg, "@")
	slashIndex := strings.Index(arg, "/")
	if slashIndex >= 0 && (atIndex == -1 || slashIndex < atIndex) {
		// There's a slash before the @ sign or there's no @ but there is a slash
		// This is only allowed if it's a parameter after the domain part
		return "", nil, "", fmt.Errorf("invalid format: unexpected '/' in certificate name part")
	}

	// Now process any parameters that appear after the domain part
	if strings.Contains(arg, "/") {
		argParts := strings.Split(arg, "/")
		domainPart = argParts[0]

		// Process any parameters after the slash
		for i := 1; i < len(argParts); i++ {
			param := argParts[i]
			if strings.HasPrefix(param, "key_type=") {
				keyType = strings.TrimPrefix(param, "key_type=")
			}
			// No logging in this function - caller should log if needed
		}
	}

	// Simple domain format (no @ symbol) - use as both cert name and domain
	if !strings.Contains(domainPart, "@") {
		// Basic validation for the domain
		if strings.ContainsAny(domainPart, "/\\") {
			return "", nil, "", fmt.Errorf("invalid domain name '%s': must not contain '/' or '\\'", domainPart)
		}
		if domainPart == "" {
			return "", nil, "", fmt.Errorf("empty domain name")
		}
		// Advanced RFC validation for DNS names
		if !manager.IsValidDNSName(domainPart) {
			return "", nil, "", fmt.Errorf("invalid domain name '%s': does not conform to DNS name standards", domainPart)
		}
		return domainPart, []string{domainPart}, keyType, nil
	}

	// Process explicit cert-name@domain format
	parts := strings.SplitN(domainPart, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", nil, "", fmt.Errorf("invalid format: expected 'cert-name@domain1,domain2,...', got '%s'", domainPart)
	}

	certName := parts[0]
	domains := []string{}
	rawDomains := strings.Split(parts[1], ",")
	for _, d := range rawDomains {
		trimmed := strings.TrimSpace(d)
		if trimmed != "" {
			// Validate the domain according to DNS standards
			if !manager.IsValidDNSName(trimmed) {
				return "", nil, "", fmt.Errorf("invalid domain name '%s': does not conform to DNS name standards", trimmed)
			}
			domains = append(domains, trimmed)
		}
	}

	if len(domains) == 0 {
		return "", nil, "", fmt.Errorf("no valid domains found after '@' in argument '%s'", domainPart)
	}

	// Basic validation for cert name
	if strings.ContainsAny(certName, "/\\") {
		return "", nil, "", fmt.Errorf("invalid certificate name '%s': must not contain '/' or '\\'", certName)
	}

	return certName, domains, keyType, nil
}

// Version information
var (
	version = "local-version" // This will be replaced during build with timestamp or actual version
)

var (
	configPath          = flag.String("config", "config.yaml", "Path to the configuration file")
	autoMode            = flag.Bool("auto", false, "Enable automatic mode using 'auto_domains' config section (handles init and renew)")
	quietMode           = flag.Bool("quiet", false, "Reduce output in auto mode (useful for cron jobs)")
	printConfigTemplate = flag.Bool("print-config-template", false, "Print a default configuration template to stdout and exit")
	debugMode           = flag.Bool("debug", false, "Enable debug logging")
	logLevel            = flag.String("log-level", "", "Set logging level (debug|info|warn|error), overrides -debug flag if specified")
	logFormat           = flag.String("log-format", "", "Set logging format (go|emoji|color|ascii), overrides -no-color and -no-emoji flags")
	showVersion         = flag.Bool("version", false, "Show version information and exit")
)

func main() {
	flag.Usage = func() {
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
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("go-acme-dns-manager %s\n", version)
		fmt.Printf("Build date: %s\n", time.Now().Format("2006-01-02"))
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// Display version on startup
	fmt.Printf("go-acme-dns-manager %s\n", version)

	// --- Logger Setup ---
	loggerLevel := manager.LogLevelInfo // Default log level
	var loggerFormat manager.LogFormat  // Will be initialized later

	// Parse log level flag if specified
	if *logLevel != "" {
		switch strings.ToLower(*logLevel) {
		case "debug":
			loggerLevel = manager.LogLevelDebug
		case "info":
			loggerLevel = manager.LogLevelInfo
		case "warn", "warning":
			loggerLevel = manager.LogLevelWarn
		case "error":
			loggerLevel = manager.LogLevelError
		default:
			fmt.Fprintf(os.Stderr, "Invalid log level: %s. Using default (info).\n", *logLevel)
		}
	} else {
		// Use the legacy flags if log-level is not specified
		if *quietMode && *autoMode {
			loggerLevel = manager.LogLevelQuiet
		} else if *debugMode {
			loggerLevel = manager.LogLevelDebug
		}
	}

	// Parse log format flag if specified
	if *logFormat != "" {
		switch strings.ToLower(*logFormat) {
		case "go":
			loggerFormat = manager.LogFormatGo
		case "emoji":
			loggerFormat = manager.LogFormatEmoji
		case "color":
			loggerFormat = manager.LogFormatColor
		case "ascii":
			loggerFormat = manager.LogFormatASCII
		default:
			fmt.Fprintf(os.Stderr, "Invalid log format: %s. Using default.\n", *logFormat)
			loggerFormat = manager.LogFormatDefault
		}
	} else {
		// Set format based on legacy flags
		loggerFormat = manager.LogFormatDefault
	}

	// Set up the logger
	manager.SetupDefaultLogger(loggerLevel, loggerFormat) // Set the default logger for the manager package
	logger := manager.GetDefaultLogger()                  // Use the configured default logger

	// Define logInfoMessage and logImportant as wrappers for the new logger
	logInfoMessage := logger.Infof
	logWarnMessage := logger.Warnf
	logDebugMessage := logger.Debugf

	// Handle print config template flag first
	if *printConfigTemplate {
		fmt.Println("# Default configuration template:")
		err := manager.GenerateDefaultConfig(os.Stdout) // Write to stdout
		if err != nil {
			logger.Errorf("Error printing config template: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// --- Config Loading ---
	absConfigPath, err := filepath.Abs(*configPath)
	if err != nil {
		logger.Errorf("Error getting absolute path for config file %s: %v", *configPath, err)
		os.Exit(1)
	}
	*configPath = absConfigPath

	// Check if config file exists, error if not (don't generate automatically)
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		logger.Errorf("Error: Configuration file not found at %s. Use -print-config-template to get a template.", *configPath)
		os.Exit(1)
	} else if err != nil {
		logger.Errorf("Error checking config file %s: %v", *configPath, err)
		os.Exit(1)
	}

	// Load configuration (file must exist at this point)
	logger.Infof("Loading configuration from %s...", *configPath)
	cfg, err := manager.LoadConfig(*configPath)
	if err != nil {
		// Check for placeholder email only, as domains list is removed/optional
		contentBytes, readErr := os.ReadFile(*configPath)
		if readErr == nil {
			content := string(contentBytes)
			if strings.Contains(content, "your-email@example.com") {
				logger.Errorf("Error: Configuration file %s still contains placeholder email. Please edit it.", *configPath)
				os.Exit(1)
			}
		}
		fmt.Fprintf(os.Stderr, "Config file error in %s: %v", *configPath, err)
		os.Exit(1)
	}
	logger.Info("Configuration loaded successfully.")

	// --- Mode Determination ---
	positionalArgs := flag.Args()
	isManualMode := len(positionalArgs) > 0
	isAutoMode := *autoMode // Use renamed flag variable

	if isManualMode && isAutoMode {
		fmt.Fprintf(os.Stderr, "Error: Cannot use -auto flag and specify certificate arguments simultaneously.\n\n")
		flag.Usage()
		os.Exit(1)
	}
	if !isManualMode && !isAutoMode {
		fmt.Fprintf(os.Stderr, "Error: No operation specified. Provide certificate arguments or use -auto flag.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// --- Account Store Initialization ---
	accountsFilePath := filepath.Join(cfg.CertStoragePath, "acme-dns-accounts.json") // Use renamed field
	logger.Infof("Loading ACME DNS accounts from %s...", accountsFilePath)
	store, err := manager.NewAccountStore(accountsFilePath)
	if err != nil {
		logger.Errorf("Error initializing account store from %s: %v", accountsFilePath, err)
		os.Exit(1)
	}

	// Log that accounts were loaded successfully
	logger.Info("ACME DNS accounts loaded successfully.")

	// --- Build List of Certificate Requests ---
	requests := []certRequest{}
	requestedNames := make(map[string]struct{}) // For duplicate name check

	if isManualMode {
		logger.Debug("Mode: Manual Specification")
		for _, arg := range positionalArgs {
			// Use the shared parsing function for all argument formats
			certName, domains, keyType, err := parseCertArg(arg)
			if err != nil {
				logger.Errorf("Error: %v", err)
				os.Exit(1)
			}

			// For simple domain format (domain used as both cert name and domain),
			// add an informational message
			if arg == certName && len(domains) == 1 && domains[0] == arg && !strings.Contains(arg, "@") {
				logDebugMessage("Interpreting argument '%s' as shorthand for '%s@%s'", arg, certName, certName)
			}

			// Log parameter information if found
			if keyType != "" {
				logDebugMessage("Found key_type parameter: %s", keyType)
			}

			if _, exists := requestedNames[certName]; exists {
				logger.Errorf("Error: Duplicate certificate name specified or implied in arguments: '%s'", certName)
				os.Exit(1)
			}
			requests = append(requests, certRequest{Name: certName, Domains: domains, KeyType: keyType})
			requestedNames[certName] = struct{}{}
		}
	} else { // Auto Mode
		logInfoMessage("Mode: Automatic") // Update log message
		if cfg.AutoDomains == nil || len(cfg.AutoDomains.Certs) == 0 {
			logInfoMessage("No certificates defined in 'auto_domains.certs' section of the config file. Nothing to do.")
			os.Exit(0)
		}
		logDebugMessage("Processing %d certificate definition(s) from config file...", len(cfg.AutoDomains.Certs))
		for name, certDef := range cfg.AutoDomains.Certs {
			// Basic validation already done in LoadConfig
			requests = append(requests, certRequest{Name: name, Domains: certDef.Domains, KeyType: certDef.KeyType})
			if certDef.KeyType != "" {
				logDebugMessage("Certificate %s will use key type: %s", name, certDef.KeyType)
			}
			// No need to check for duplicate names here as map keys are unique
		}
	}

	// --- Pre-Check for Conflicts and Determine Actions ---
	logDebugMessage("Performing pre-checks for requested certificates...")
	type requestTask struct {
		Request certRequest
		Action  string // "init", "renew", "skip"
	}
	tasks := []requestTask{}
	renewalThreshold := cfg.GetRenewalThreshold() // Get duration from config/default

	for _, req := range requests {
		logDebugMessage("Checking certificate: %s (%v)", req.Name, req.Domains)
		action := "init" // Default action is init
		// skip := false // Removed unused variable

		metaPath := filepath.Join(cfg.CertStoragePath, "certificates", req.Name+".json") // Use renamed field
		certPath := filepath.Join(cfg.CertStoragePath, "certificates", req.Name+".crt")  // Use renamed field

		if _, err := os.Stat(metaPath); err == nil {
			// Metadata exists, potential renew
			action = "renew"
			logger.Debugf("Existing metadata found (%s). Checking domains and expiry.", metaPath)

			// Load existing metadata to check domains
			existingCertData, err := manager.LoadCertificateResource(cfg, req.Name) // Assuming LoadCertificateResource exists
			if err != nil {
				logger.Errorf("Error loading existing certificate metadata for '%s' from %s: %v", req.Name, metaPath, err)
				os.Exit(1)
			}

			// Simplified Check: Compare only the primary domain.
			// A full SAN list comparison seems problematic with the loaded resource struct.
			// This ensures the main domain matches the existing cert.
			if len(req.Domains) > 0 && req.Domains[0] != existingCertData.Domain {
				logger.Errorf("Error: Primary domain mismatch for certificate '%s'.\n  Requested primary: %s\n  Existing primary (%s): %s\nPlease use a different certificate name or manually remove the old files.",
					req.Name, req.Domains[0], metaPath, existingCertData.Domain)
				os.Exit(1)
			} else if len(req.Domains) == 0 {
				// Should not happen due to earlier parsing checks, but safety first
				logger.Errorf("Internal Error: Empty domain list for certificate request '%s'", req.Name)
			} else {
				logger.Debugf("Primary domain '%s' matches existing certificate.", req.Domains[0])

				// Attempt to load and parse the actual certificate to get the SAN list
				certPath := filepath.Join(cfg.CertStoragePath, "certificates", req.Name+".crt")
				certBytes, err := os.ReadFile(certPath)
				if err == nil {
					block, _ := pem.Decode(certBytes)
					if block != nil {
						cert, err := x509.ParseCertificate(block.Bytes)
						if err == nil {
							// Compare requested domains with certificate SAN list
							// Create maps for easier comparison
							existingDomainsMap := make(map[string]bool)
							for _, domain := range cert.DNSNames {
								existingDomainsMap[domain] = true
							}

							requestedDomainsMap := make(map[string]bool)
							for _, domain := range req.Domains {
								requestedDomainsMap[domain] = true
							}

							// Check for differences
							var missingDomains, extraDomains []string
							for _, domain := range req.Domains {
								if !existingDomainsMap[domain] {
									missingDomains = append(missingDomains, domain)
								}
							}

							for _, domain := range cert.DNSNames {
								if !requestedDomainsMap[domain] {
									extraDomains = append(extraDomains, domain)
								}
							}

							if len(missingDomains) > 0 || len(extraDomains) > 0 {
								logger.Warnf("Domain list differences detected for certificate '%s':", req.Name)
								if len(missingDomains) > 0 {
									logger.Infof("    - New domains that will be added: %v", missingDomains)
									// Force renewal when domains are missing from the certificate
									logger.Infof("    - Will force renewal to include all requested domains.")
									// Make sure we don't skip this certificate even if not expiring soon
									if action != "renew" {
										logger.Debugf("    - Previous action was '%s'", action)
										action = "renew"
										logger.Infof("    - Changed action to 'renew' due to missing domains.")
									}
								}
								if len(extraDomains) > 0 {
									logger.Infof("    - Domains in existing cert that will be removed: %v", extraDomains)
								}
								logger.Infof("  If this is not intended, please use a different certificate name.")
							} else {
								logger.Debugf("All domains match between requested domains and existing certificate.")
							}
						} else {
							logger.Warnf("Could not parse certificate from %s: %v. Skipping SAN comparison.", certPath, err)
						}
					} else {
						logger.Warnf("Failed to decode PEM block from %s. Skipping SAN comparison.", certPath)
					}
				} else {
					logger.Warnf("Could not read certificate file %s: %v. Skipping SAN comparison.", certPath, err)
				}

			}

			// If in auto mode, check expiry date
			if isAutoMode { // Use renamed flag variable
				certBytes, err := os.ReadFile(certPath)
				if err != nil {
					logger.Warnf("Could not read existing certificate file %s for expiry check: %v. Proceeding with renewal.", certPath, err)
				} else {
					block, _ := pem.Decode(certBytes)
					if block == nil {
						logger.Warnf("Failed to decode PEM block from %s. Proceeding with renewal.", certPath)
					} else {
						cert, err := x509.ParseCertificate(block.Bytes)
						if err != nil {
							logger.Warnf("Failed to parse certificate from %s: %v. Proceeding with renewal.", certPath, err)
						} else {
							timeLeft := time.Until(cert.NotAfter)
							logger.Debugf("Certificate expires on %s (%v remaining). Renewal threshold is %v.", cert.NotAfter.Format(time.RFC1123), timeLeft.Round(time.Hour), renewalThreshold)

							// Check for domain differences that would force renewal
							domainChanges := false
							// We need to load the cert again to check its domains
							cert2, err := x509.ParseCertificate(block.Bytes)
							if err == nil {
								// Check if all requested domains exist in the certificate
								existingDomainsMap := make(map[string]bool)
								for _, domain := range cert2.DNSNames {
									existingDomainsMap[domain] = true
								}

								for _, domain := range req.Domains {
									if !existingDomainsMap[domain] {
										// Domain is missing, need to force renewal
										domainChanges = true
										break
									}
								}
							}

							// If expiration is OK and no domain changes needed, skip renewal
							if timeLeft > renewalThreshold && !domainChanges {
								logger.Debugf("Skipping renewal: Certificate is not within the renewal threshold and no domain changes needed.")
								action = "skip" // Mark as skip
								logger.Infof("Certificate '%s' doesn't need renewal - will be skipped", req.Name)
							} else if timeLeft <= renewalThreshold {
								logger.Warnf("Certificate is within renewal threshold. Proceeding with renewal.")
							} else if domainChanges {
								logger.Debugf("Certificate will be renewed due to domain changes")
							}
						}
					}
				}
			}

		} else if !os.IsNotExist(err) {
			// Error checking file other than not found
			logger.Errorf("Error checking certificate metadata file %s: %v", metaPath, err)
			os.Exit(1)
		} else {
			logger.Debugf("No existing metadata found (%s). Action set to 'init'.", metaPath)
		}

		// Always add the task for informational purposes
		task := requestTask{Request: req, Action: action}
		tasks = append(tasks, task)

		// Log when we're skipping a certificate
		if action == "skip" {
			logger.Infof("Certificate '%s' doesn't need renewal - skipping processing", req.Name)
		}
	} // End pre-check loop
	// Filter out certificates marked for skipping
	var processingTasks []requestTask
	logDebugMessage("Filtering tasks marked for skipping:")
	for _, task := range tasks {
		logDebugMessage("Checking task for certificate '%s', Action='%s'", task.Request.Name, task.Action)
		if task.Action != "skip" {
			processingTasks = append(processingTasks, task)
			logDebugMessage("  - Added to processing list: Certificate='%s', Action='%s'", task.Request.Name, task.Action)
		} else {
			logDebugMessage("  - Skipped from processing: Certificate='%s'", task.Request.Name)
		}
	}

	logDebugMessage("After filtering: %d of %d tasks will be processed", len(processingTasks), len(tasks))

	if len(processingTasks) == 0 {
		logger.Info("No certificates require processing.")
		os.Exit(0)
	}

	logDebugMessage("Pre-checks complete. Processing %d certificate task(s)...", len(processingTasks))
	// Define a struct to track required CNAME changes
	type requiredCNAME struct {
		Domain      string
		CNAMERecord string
		Target      string
	}

	// List to collect required CNAME records
	var requiredCNAMEs []requiredCNAME

	// --- Process Tasks (ACME DNS Verification & Lego Execution) ---
	anyFailure := false
	for _, task := range processingTasks {
		certName := task.Request.Name
		domains := task.Request.Domains
		action := task.Action
		taskHasFailure := false // Track failures for this specific task only

		logInfoMessage("Processing Task: Action=%s, CertName=%s, Domains=%v ---", action, certName, domains)

		// 1. Verify/Register ACME DNS for all domains in this group
		needsManualUpdate := false
		logInfoMessage("Verifying/Registering ACME DNS accounts for %d domain(s)...", len(domains))
		// For wildcard domains, we need to keep track of which base domains we've already validated
		checkedBaseDomains := make(map[string]bool)

		for _, domain := range domains {
			// Check if this is a wildcard domain and if we've already validated the base domain
			baseDomain := manager.GetBaseDomain(domain)
			if strings.HasPrefix(domain, "*.") && checkedBaseDomains[baseDomain] {
				// We've already validated the base domain CNAME, skip redundant checks
				logInfoMessage("Using already verified CNAME for %s based on %s", domain, baseDomain)
				continue
			}

			account, exists := store.GetAccount(domain)

			if !exists {
				logInfoMessage("Registering ACME DNS for %s...", domain)
				newAccount, err := manager.RegisterNewAccount(cfg, store, domain)
				if err != nil {
					logger.Errorf("ERROR registering new acme-dns account for %s: %v", domain, err)
					taskHasFailure = true // Set the task-specific failure flag
					anyFailure = true     // Also set the global flag for exit code
					break                 // Stop processing this cert group if registration fails
				}

				// Instead of printing immediately, collect for final report
				baseDomain := manager.GetBaseDomain(domain)
				cnameRecord := fmt.Sprintf("_acme-challenge.%s", baseDomain)
				requiredCNAMEs = append(requiredCNAMEs, requiredCNAME{
					Domain:      domain,
					CNAMERecord: cnameRecord,
					Target:      newAccount.FullDomain,
				})

				needsManualUpdate = true
				continue
			}

			// Account exists, verify CNAME
			logInfoMessage("Verifying CNAME for %s...", domain)
			cnameValid, err := manager.VerifyCnameRecord(cfg, domain, account.FullDomain)

			// If this is a base domain and the CNAME is valid, mark it as checked
			if cnameValid && !strings.HasPrefix(domain, "*.") {
				checkedBaseDomains[domain] = true
			}
			if err != nil {
				logger.Errorf("Error verifying CNAME record for %s: %v. Treating as invalid.", domain, err)

				// Instead of printing immediately, collect for final report
				baseDomain := manager.GetBaseDomain(domain)
				cnameRecord := fmt.Sprintf("_acme-challenge.%s", baseDomain)
				requiredCNAMEs = append(requiredCNAMEs, requiredCNAME{
					Domain:      domain,
					CNAMERecord: cnameRecord,
					Target:      account.FullDomain,
				})

				needsManualUpdate = true
			} else if !cnameValid {
				logger.Errorf("CNAME record for %s is missing or invalid.", domain)

				// Instead of printing immediately, collect for final report
				baseDomain := manager.GetBaseDomain(domain)
				cnameRecord := fmt.Sprintf("_acme-challenge.%s", baseDomain)
				requiredCNAMEs = append(requiredCNAMEs, requiredCNAME{
					Domain:      domain,
					CNAMERecord: cnameRecord,
					Target:      account.FullDomain,
				})

				needsManualUpdate = true
			} else {
				logInfoMessage("CNAME record for %s is valid.", domain)
			}
		} // End domain loop for ACME DNS

		if taskHasFailure {
			logWarnMessage("Skipping Lego operation for '%s' due to errors with this certificate.", certName)
			continue // Move to the next task
		}

		if needsManualUpdate {
			logWarnMessage("Manual DNS CNAME updates required for certificate '%s'.", certName)
			anyFailure = true // Treat as failure for overall exit code
			continue          // Move to the next task
		}

		logInfoMessage("All domains for '%s' have valid ACME DNS configurations.", certName)

		// 2. Run Lego action
		logInfoMessage("Proceeding with Lego action '%s' for certificate '%s'...", action, certName)
		keyType := task.Request.KeyType
		if keyType != "" {
			logInfoMessage("Using specified key type for certificate: %s", keyType)
		}
		err = manager.RunLego(cfg, store, action, certName, domains, keyType) // Pass certName and keyType
		if err != nil {
			logger.Errorf("ERROR: Lego operation failed for certificate '%s': %v", certName, err)
			taskHasFailure = true // Mark this specific task as failed
			anyFailure = true     // Also set the global flag for the final exit code
			// Continue to next task even if one fails? Or stop? Let's continue for now.
		} else {
			logger.Infof("Lego operation successful for certificate '%s'.", certName)
		}

	} // End task processing loop
	// --- Final Status ---
	if len(requiredCNAMEs) > 0 {
		// Print DNS changes directly to stdout as these are important user-facing information
		fmt.Println("\n===== REQUIRED DNS CHANGES =====")
		fmt.Println("Add the following CNAME records to your DNS:")
		fmt.Println()

		// Create maps to group by CNAME record AND by target
		cnameMap := make(map[string]map[string][]string) // Map[CNAMERecord]Map[Target][]Domains

		// First organize all records
		for _, cname := range requiredCNAMEs {
			// Initialize the map for this CNAME record if it doesn't exist
			if _, exists := cnameMap[cname.CNAMERecord]; !exists {
				cnameMap[cname.CNAMERecord] = make(map[string][]string)
			}

			// Add this domain to the appropriate target group for this CNAME record
			cnameMap[cname.CNAMERecord][cname.Target] = append(
				cnameMap[cname.CNAMERecord][cname.Target],
				cname.Domain,
			)
		}

		// Now print each unique CNAME record with its domains grouped by target
		for cnameRecord, targetGroups := range cnameMap {
			for target, domains := range targetGroups {
				// Create a proper comment showing all domains using this record
				commentParts := []string{}
				for _, domain := range domains {
					if strings.HasPrefix(domain, "*.") {
						commentParts = append(commentParts, domain+" (wildcard)")
					} else {
						commentParts = append(commentParts, domain)
					}
				}
				comment := strings.Join(commentParts, ", ")

				// Print in BIND format with comment
				fmt.Printf("; %s\n", comment)
				fmt.Printf("%s. IN CNAME %s.\n\n", cnameRecord, target)
			}
		}

		fmt.Println("Please make these DNS changes and run the command again.")
		anyFailure = false // Reset failure flag for manual intervention
		os.Exit(1)         // Exit with error code for manual intervention
	}

	if anyFailure {
		logger.Errorf("One or more operations failed or require manual intervention.")
		os.Exit(1)
	}
	logInfoMessage("\nOperation completed successfully.")
}
