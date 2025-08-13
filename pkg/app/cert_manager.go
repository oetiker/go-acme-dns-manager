package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/common"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// LegoRunnerFunc is a function type that matches the signature of manager.RunLegoWithStore
type LegoRunnerFunc func(cfg *manager.Config, store interface{}, action string, certName string, domains []string, keyType string) error

// DefaultLegoRunner is the default implementation that calls the real ACME server
var DefaultLegoRunner LegoRunnerFunc = manager.RunLegoWithStore

// CertificateManager handles certificate operations with clean separation of concerns
type CertificateManager struct {
	config       *manager.Config
	logger       common.LoggerInterface
	accountStore interface{}
	legoRunner   LegoRunnerFunc
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(config *manager.Config, logger common.LoggerInterface) (*CertificateManager, error) {
	accountsFilePath := filepath.Join(config.CertStoragePath, "acme-dns-accounts.json")
	logger.Infof("Loading ACME DNS accounts from %s...", accountsFilePath)

	// Initialize the account store
	store, err := manager.NewAccountStore(accountsFilePath)
	if err != nil {
		return nil, fmt.Errorf("creating account store: %w", err)
	}

	logger.Info("ACME DNS accounts loaded successfully.")

	return &CertificateManager{
		config:       config,
		logger:       logger,
		accountStore: store,
		legoRunner:   DefaultLegoRunner,
	}, nil
}

// SetLegoRunner sets a custom Lego runner function (mainly for testing)
func (cm *CertificateManager) SetLegoRunner(runner LegoRunnerFunc) {
	cm.legoRunner = runner
}

// CertRequest represents a certificate request
type CertRequest struct {
	Name    string
	Domains []string
	KeyType string
}

// ProcessManualMode handles manual certificate requests from command line arguments
func (cm *CertificateManager) ProcessManualMode(ctx context.Context, args []string) error {
	cm.logger.Debug("Mode: Manual Specification")

	requests, err := cm.parseManualRequests(args)
	if err != nil {
		return err
	}

	return cm.processRequests(ctx, requests)
}

// ProcessAutoMode handles automatic certificate processing from config
func (cm *CertificateManager) ProcessAutoMode(ctx context.Context) error {
	cm.logger.Info("Mode: Automatic")

	if cm.config.AutoDomains == nil || len(cm.config.AutoDomains.Certs) == 0 {
		cm.logger.Info("No certificates defined in 'auto_domains.certs' section of the config file. Nothing to do.")
		return nil
	}

	requests := cm.parseAutoRequests()
	return cm.processRequests(ctx, requests)
}

// parseManualRequests parses command line arguments into certificate requests
func (cm *CertificateManager) parseManualRequests(args []string) ([]CertRequest, error) {
	var requests []CertRequest
	requestedNames := make(map[string]struct{})

	for _, arg := range args {
		certName, domains, keyType, err := manager.ParseCertArg(arg)
		if err != nil {
			return nil, fmt.Errorf("parsing argument %s: %w", arg, err)
		}

		// Check for duplicates
		if _, exists := requestedNames[certName]; exists {
			return nil, fmt.Errorf("duplicate certificate name specified: '%s'", certName)
		}

		// Log parameter information
		if keyType != "" {
			cm.logger.Debugf("Found key_type parameter: %s", keyType)
		}

		requests = append(requests, CertRequest{
			Name:    certName,
			Domains: domains,
			KeyType: keyType,
		})
		requestedNames[certName] = struct{}{}
	}

	return requests, nil
}

// parseAutoRequests parses automatic requests from config
func (cm *CertificateManager) parseAutoRequests() []CertRequest {
	var requests []CertRequest

	cm.logger.Debugf("Processing %d certificate definition(s) from config file...", len(cm.config.AutoDomains.Certs))

	for name, certDef := range cm.config.AutoDomains.Certs {
		requests = append(requests, CertRequest{
			Name:    name,
			Domains: certDef.Domains,
			KeyType: certDef.KeyType,
		})

		if certDef.KeyType != "" {
			cm.logger.Debugf("Certificate %s will use key type: %s", name, certDef.KeyType)
		}
	}

	return requests
}

// processRequests processes a list of certificate requests
func (cm *CertificateManager) processRequests(ctx context.Context, requests []CertRequest) error {
	cm.logger.Debugf("Performing pre-checks for %d requested certificates...", len(requests))

	renewalThreshold := cm.config.GetRenewalThreshold()

	for _, req := range requests {
		if err := cm.processRequest(ctx, req, renewalThreshold); err != nil {
			return fmt.Errorf("processing certificate %s: %w", req.Name, err)
		}
	}

	return nil
}

// processRequest processes a single certificate request
func (cm *CertificateManager) processRequest(ctx context.Context, req CertRequest, renewalThreshold interface{}) error {
	cm.logger.Debugf("Processing certificate: %s (%v)", req.Name, req.Domains)

	// Determine action needed (init, renew, skip)
	action, err := cm.determineAction(req, renewalThreshold)
	if err != nil {
		return err
	}

	cm.logger.Infof("Certificate %s requires action: %s", req.Name, action)

	// Execute the action
	switch action {
	case "init":
		return cm.initCertificate(ctx, req)
	case "renew":
		return cm.renewCertificate(ctx, req)
	case "skip":
		cm.logger.Infof("Certificate %s is up to date, skipping", req.Name)
		return nil
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// determineAction determines what action is needed for a certificate
func (cm *CertificateManager) determineAction(req CertRequest, renewalThreshold interface{}) (string, error) {
	// Convert renewalThreshold to time.Duration
	threshold, ok := renewalThreshold.(time.Duration)
	if !ok {
		return "", fmt.Errorf("invalid renewal threshold type: %T", renewalThreshold)
	}

	// Check if certificate metadata exists - this determines if it's a new cert or renewal
	certPath := filepath.Join(cm.config.CertStoragePath, "certificates", req.Name+".crt")
	metadataPath := filepath.Join(cm.config.CertStoragePath, "certificates", req.Name+".json")

	// If metadata file doesn't exist, it's a new certificate
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		cm.logger.Debugf("Certificate metadata not found at %s - initializing new certificate", metadataPath)
		return "init", nil
	} else if err != nil {
		return "", fmt.Errorf("checking certificate metadata %s: %w", metadataPath, err)
	}

	// If certificate file doesn't exist, it's a new certificate
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		cm.logger.Debugf("Certificate file not found at %s - initializing new certificate", certPath)
		return "init", nil
	} else if err != nil {
		return "", fmt.Errorf("checking certificate file %s: %w", certPath, err)
	}

	// Certificate exists, check if it needs renewal
	needsRenewal, reason, err := manager.CertificateNeedsRenewal(certPath, req.Domains, threshold)
	if err != nil {
		cm.logger.Warnf("Error checking certificate renewal status: %v", err)
		// If we can't check the certificate, assume it needs renewal
		return "renew", nil
	}

	if needsRenewal {
		cm.logger.Infof("Certificate %s needs renewal: %s", req.Name, reason)
		return "renew", nil
	}

	// Certificate exists and doesn't need renewal
	cm.logger.Infof("Certificate %s is valid and doesn't need renewal", req.Name)
	return "skip", nil
}

// initCertificate initializes a new certificate
func (cm *CertificateManager) initCertificate(ctx context.Context, req CertRequest) error {
	cm.logger.Infof("Initializing certificate %s for domains %v", req.Name, req.Domains)

	// Check if we were asked to shutdown
	if common.IsContextCanceled(ctx) {
		return common.GetContextError(ctx, "certificate initialization")
	}

	// Call the manager's RunLego function to obtain the certificate
	err := cm.legoRunner(cm.config, cm.accountStore, "init", req.Name, req.Domains, req.KeyType)
	if err != nil {
		// Check if this is just DNS setup needed (not really an error)
		if errors.Is(err, manager.ErrDNSSetupNeeded) {
			// DNS setup instructions were already shown, this is a normal exit
			return err // Return the error as-is to bubble up
		}
		return fmt.Errorf("failed to initialize certificate %s: %w", req.Name, err)
	}

	cm.logger.Infof("Certificate %s initialized successfully", req.Name)
	return nil
}

// renewCertificate renews an existing certificate
func (cm *CertificateManager) renewCertificate(ctx context.Context, req CertRequest) error {
	cm.logger.Infof("Renewing certificate %s for domains %v", req.Name, req.Domains)

	// Check if we were asked to shutdown
	if common.IsContextCanceled(ctx) {
		return common.GetContextError(ctx, "certificate renewal")
	}

	// Call the manager's RunLego function to renew the certificate
	err := cm.legoRunner(cm.config, cm.accountStore, "renew", req.Name, req.Domains, req.KeyType)
	if err != nil {
		return fmt.Errorf("failed to renew certificate %s: %w", req.Name, err)
	}

	cm.logger.Infof("Certificate %s renewed successfully", req.Name)
	return nil
}
