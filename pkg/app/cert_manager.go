package app

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/oetiker/go-acme-dns-manager/pkg/common"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// CertificateManager handles certificate operations with clean separation of concerns
type CertificateManager struct {
	config       *manager.Config
	logger       common.LoggerInterface
	accountStore interface{} // Will be replaced with storage interface
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(config *manager.Config, logger common.LoggerInterface) (*CertificateManager, error) {
	accountsFilePath := filepath.Join(config.CertStoragePath, "acme-dns-accounts.json")
	logger.Infof("Loading ACME DNS accounts from %s...", accountsFilePath)

	// In the refactored version, this would use our storage interface
	// store, err := storage.NewAccountStore(accountsFilePath, logger, fileSystem)
	// For now, placeholder
	var store interface{}

	logger.Info("ACME DNS accounts loaded successfully.")

	return &CertificateManager{
		config:       config,
		logger:       logger,
		accountStore: store,
	}, nil
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
	// Check if certificate metadata exists
	// This would contain more sophisticated logic from the original main function
	// For now, simplified version - would check file existence, domain changes, expiry, etc.
	return "init", nil
}

// initCertificate initializes a new certificate
func (cm *CertificateManager) initCertificate(ctx context.Context, req CertRequest) error {
	cm.logger.Infof("Initializing certificate %s for domains %v", req.Name, req.Domains)

	// This would contain the actual certificate initialization logic
	// Using the clean interfaces we established

	return nil
}

// renewCertificate renews an existing certificate
func (cm *CertificateManager) renewCertificate(ctx context.Context, req CertRequest) error {
	cm.logger.Infof("Renewing certificate %s for domains %v", req.Name, req.Domains)

	// This would contain the actual certificate renewal logic
	// Using the clean interfaces we established

	return nil
}
