package manager

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/acmedns"
	"github.com/go-acme/lego/v4/registration"
)

// ErrDNSSetupNeeded is returned when DNS configuration is required.
// This is not really an error but a normal part of the setup flow.
var ErrDNSSetupNeeded = errors.New("DNS configuration needed")

// RunLegoWithStore is a wrapper function that accepts interface{} for the store parameter
// and performs the type assertion internally. This allows external packages to call RunLego
// without needing to import the unexported accountStore type.
func RunLegoWithStore(cfg *Config, store interface{}, action string, certName string, domainsToProcess []string, keyType string) error {
	accountStore, ok := store.(*accountStore)
	if !ok {
		return fmt.Errorf("invalid store type: expected *accountStore, got %T", store)
	}
	return RunLego(cfg, accountStore, action, certName, domainsToProcess, keyType)
}

// PreCheckAcmeDNSWithStore is a wrapper function that accepts interface{} for the store parameter
// and performs the type assertion internally. This allows external packages to call PreCheckAcmeDNS
// without needing to import the unexported accountStore type.
func PreCheckAcmeDNSWithStore(cfg *Config, store interface{}, domains []string) ([]DNSSetupInfo, error) {
	accountStore, ok := store.(*accountStore)
	if !ok {
		return nil, fmt.Errorf("invalid store type: expected *accountStore, got %T", store)
	}
	return PreCheckAcmeDNS(cfg, accountStore, domains)
}

// PreCheckAcmeDNSWithStoreAndResolver is a wrapper that accepts both store as interface{} and a DNS resolver
// This allows external packages to inject a custom DNS resolver for testing
func PreCheckAcmeDNSWithStoreAndResolver(cfg *Config, store interface{}, domains []string, resolver DNSResolver) ([]DNSSetupInfo, error) {
	accountStore, ok := store.(*accountStore)
	if !ok {
		return nil, fmt.Errorf("invalid store type: expected *accountStore, got %T", store)
	}
	return PreCheckAcmeDNSWithResolver(cfg, accountStore, domains, resolver)
}

// PreCheckAcmeDNSWithResolver is a version that allows injection of a DNS resolver for testing
func PreCheckAcmeDNSWithResolver(cfg *Config, store *accountStore, domains []string, resolver DNSResolver) ([]DNSSetupInfo, error) {
	// Use a map to avoid duplicate CNAME instructions
	cnameMap := make(map[string]string)

	// First pass: Register any missing ACME-DNS accounts
	for _, domain := range domains {
		baseDomain := GetBaseDomain(domain)
		_, exists := store.GetAccount(baseDomain)
		if !exists {
			// Try wildcard version
			wildcardDomain := "*." + baseDomain
			_, exists = store.GetAccount(wildcardDomain)
			if !exists {
				// No account exists, register a new one with acme-dns
				DefaultLogger.Infof("No ACME-DNS account found for domain %s, registering new account...", domain)
				newAccount, err := RegisterNewAccount(cfg, store, domain)
				if err != nil {
					return nil, fmt.Errorf("failed to register ACME-DNS account for domain %s: %w", domain, err)
				}

				// Save the updated account store immediately
				if err := store.SaveAccounts(); err != nil {
					return nil, fmt.Errorf("failed to save ACME-DNS accounts: %w", err)
				}

				challengeDomain := "_acme-challenge." + GetBaseDomain(domain)
				cnameMap[challengeDomain] = newAccount.FullDomain
			}
		}
	}

	// Second pass: Check CNAME records for all domains using provided resolver
	for _, domain := range domains {
		baseDomain := GetBaseDomain(domain)
		account, exists := store.GetAccount(baseDomain)
		if !exists {
			// Try wildcard version
			wildcardDomain := "*." + baseDomain
			account, exists = store.GetAccount(wildcardDomain)
		}

		if exists {
			// Check CNAME silently (no logging)
			challengeDomain := "_acme-challenge." + GetBaseDomain(domain)
			expectedTarget := strings.TrimSuffix(account.FullDomain, ".")

			isValid, err := VerifyWithResolver(resolver, challengeDomain, expectedTarget)
			if err != nil {
				return nil, fmt.Errorf("DNS verification failed for %s: %w", domain, err)
			}

			if !isValid {
				// Add to map (automatically handles duplicates)
				cnameMap[challengeDomain] = account.FullDomain
			}
		}
	}

	// Convert map to slice of DNSSetupInfo if any setup is needed
	if len(cnameMap) > 0 {
		var setupInfo []DNSSetupInfo
		for challenge, target := range cnameMap {
			setupInfo = append(setupInfo, DNSSetupInfo{
				ChallengeDomain: challenge,
				TargetDomain:    target,
			})
		}
		return setupInfo, nil
	}

	return nil, nil
}

// DNSSetupInfo contains information about required DNS setup
type DNSSetupInfo struct {
	ChallengeDomain string
	TargetDomain    string
}

// PreCheckAcmeDNS ensures all domains have ACME-DNS accounts and valid CNAME records
// Returns DNS setup information if setup is needed, nil if all domains are ready
func PreCheckAcmeDNS(cfg *Config, store *accountStore, domains []string) ([]DNSSetupInfo, error) {
	// Use a map to avoid duplicate CNAME instructions
	cnameMap := make(map[string]string)

	// First pass: Register any missing ACME-DNS accounts
	for _, domain := range domains {
		baseDomain := GetBaseDomain(domain)
		_, exists := store.GetAccount(baseDomain)
		if !exists {
			// Try wildcard version
			wildcardDomain := "*." + baseDomain
			_, exists = store.GetAccount(wildcardDomain)
			if !exists {
				// No account exists, register a new one with acme-dns
				DefaultLogger.Infof("No ACME-DNS account found for domain %s, registering new account...", domain)
				newAccount, err := RegisterNewAccount(cfg, store, domain)
				if err != nil {
					return nil, fmt.Errorf("failed to register ACME-DNS account for domain %s: %w", domain, err)
				}

				// Save the updated account store immediately
				if err := store.SaveAccounts(); err != nil {
					return nil, fmt.Errorf("failed to save ACME-DNS accounts: %w", err)
				}

				challengeDomain := "_acme-challenge." + GetBaseDomain(domain)
				cnameMap[challengeDomain] = newAccount.FullDomain
			}
		}
	}

	// Second pass: Check CNAME records for all domains
	for _, domain := range domains {
		baseDomain := GetBaseDomain(domain)
		account, exists := store.GetAccount(baseDomain)
		if !exists {
			// Try wildcard version
			wildcardDomain := "*." + baseDomain
			account, exists = store.GetAccount(wildcardDomain)
		}

		if exists {
			// Create resolver based on config
			var resolver DNSResolver
			if cfg.DnsResolver != "" {
				// Use custom DNS resolver with specified IP
				nsAddr := cfg.DnsResolver
				if !strings.Contains(nsAddr, ":") {
					nsAddr += ":53"
				}
				resolver = &DefaultDNSResolver{
					Resolver: &net.Resolver{
						PreferGo: true,
						Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
							d := net.Dialer{
								Timeout: time.Second * 10,
							}
							return d.DialContext(ctx, network, nsAddr)
						},
					},
				}
			} else {
				// Use system default resolver
				resolver = &DefaultDNSResolver{
					Resolver: net.DefaultResolver,
				}
			}

			// Check CNAME silently (no logging)
			challengeDomain := "_acme-challenge." + GetBaseDomain(domain)
			expectedTarget := strings.TrimSuffix(account.FullDomain, ".")

			isValid, err := VerifyWithResolver(resolver, challengeDomain, expectedTarget)
			if err != nil {
				return nil, fmt.Errorf("DNS verification failed for %s: %w", domain, err)
			}

			if !isValid {
				// Add to map (automatically handles duplicates)
				cnameMap[challengeDomain] = account.FullDomain
			}
		}
	}

	// Convert map to slice of DNSSetupInfo if any setup is needed
	if len(cnameMap) > 0 {
		var setupInfo []DNSSetupInfo
		for challenge, target := range cnameMap {
			setupInfo = append(setupInfo, DNSSetupInfo{
				ChallengeDomain: challenge,
				TargetDomain:    target,
			})
		}
		return setupInfo, nil
	}

	return nil, nil
}

// DisplayDNSInstructions shows DNS setup instructions in a sorted, deduplicated format
func DisplayDNSInstructions(setupInfo []DNSSetupInfo) {
	// Sort by challenge domain for consistent output
	sortedInfo := make([]DNSSetupInfo, len(setupInfo))
	copy(sortedInfo, setupInfo)

	// Simple bubble sort for clarity (small lists)
	for i := 0; i < len(sortedInfo)-1; i++ {
		for j := 0; j < len(sortedInfo)-i-1; j++ {
			if sortedInfo[j].ChallengeDomain > sortedInfo[j+1].ChallengeDomain {
				sortedInfo[j], sortedInfo[j+1] = sortedInfo[j+1], sortedInfo[j]
			}
		}
	}

	// Use Warn level so it shows even in quiet mode (these are required actions)
	DefaultLogger.Warn("")
	DefaultLogger.Warn("===== REQUIRED DNS CHANGES =====")
	DefaultLogger.Warn("Add the following CNAME record(s) to your DNS:")
	DefaultLogger.Warn("")
	for _, info := range sortedInfo {
		DefaultLogger.Warnf("    %s. IN CNAME %s.", info.ChallengeDomain, info.TargetDomain)
	}
	DefaultLogger.Warn("")
	DefaultLogger.Warn("=================================")
	DefaultLogger.Warn("")
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RunLego performs the certificate obtain or renew operation.
// Accepts config, account store, action, the certificate name, the domains list, and optional key type.
// Exported function
func RunLego(cfg *Config, store *accountStore, action string, certName string, domainsToProcess []string, keyType string) error {
	// Validate domainsToProcess ische not empty (should be caught by main, but good practice)
	if len(domainsToProcess) == 0 {
		return fmt.Errorf("RunLego called with empty domains list")
	}

	// Pre-check ACME-DNS setup for all domains BEFORE initializing Lego
	if action == "init" {
		setupInfo, err := PreCheckAcmeDNS(cfg, store, domainsToProcess)
		if err != nil {
			return err
		}
		if setupInfo != nil {
			// DNS setup is needed, display instructions and return
			DisplayDNSInstructions(setupInfo)
			return ErrDNSSetupNeeded
		}
	}

	DefaultLogger.Info("Initializing Lego client...")

	user, userErr := createOrLoadUser(cfg)
	if userErr != nil {
		return fmt.Errorf("failed to create/load ACME user: %w", userErr)
	}

	// Setup Lego config
	legoConfig := lego.NewConfig(user)
	legoConfig.CADirURL = cfg.AcmeServer

	// Set key type, using provided value, or fall back to default
	certKeyType := DefaultKeyType
	if keyType != "" && isValidKeyType(keyType) {
		certKeyType = keyType
		DefaultLogger.Infof("Using specified key type: %s", certKeyType)
	} else {
		DefaultLogger.Infof("Using default key type: %s", certKeyType)
	}

	// Map our key types to Lego's certcrypto constants
	var legoKeyType certcrypto.KeyType
	switch certKeyType {
	case "rsa2048":
		legoKeyType = certcrypto.RSA2048
	case "rsa3072":
		legoKeyType = certcrypto.RSA3072
	case "rsa4096":
		legoKeyType = certcrypto.RSA4096
	case "ec256":
		legoKeyType = certcrypto.EC256
	case "ec384":
		legoKeyType = certcrypto.EC384
	default:
		// Default to RSA2048 if we don't have a mapping (shouldn't happen due to validation)
		legoKeyType = certcrypto.RSA2048
	}

	legoConfig.Certificate.KeyType = legoKeyType
	// Use timeouts from config
	legoConfig.Certificate.Timeout = cfg.ChallengeTimeout
	if legoConfig.HTTPClient == nil {
		legoConfig.HTTPClient = &http.Client{}
	}
	legoConfig.HTTPClient.Timeout = cfg.HTTPTimeout

	// Create Lego client
	client, clientErr := lego.NewClient(legoConfig)
	if clientErr != nil {
		return fmt.Errorf("failed to create Lego client: %w", clientErr)
	}

	// This ensures only DNS-01 is used and prevents Lego from attempting other challenge types
	client.Challenge.Remove(challenge.HTTP01)
	client.Challenge.Remove(challenge.TLSALPN01)

	// Setup acme-dns provider
	// The provider reads ACME_DNS_API_BASE and ACME_DNS_STORAGE_PATH from env vars.
	DefaultLogger.Info("Configuring ACME DNS provider...")

	// Set the environment variables required by the acme-dns provider
	DefaultLogger.Infof("Setting ACME_DNS_API_BASE=%s", cfg.AcmeDnsServer)
	if setErr := os.Setenv("ACME_DNS_API_BASE", cfg.AcmeDnsServer); setErr != nil {
		return fmt.Errorf("failed to set ACME_DNS_API_BASE env var: %w", setErr)
	}

	// The acmedns provider uses the storage path to read the credentials from the JSON file
	DefaultLogger.Infof("Setting ACME_DNS_STORAGE_PATH=%s", store.filePath)
	if setErr := os.Setenv("ACME_DNS_STORAGE_PATH", store.filePath); setErr != nil {
		return fmt.Errorf("failed to set ACME_DNS_STORAGE_PATH env var: %w", setErr)
	}

	// Create the provider using our configured environment variables
	var provider *acmedns.DNSProvider
	var providerErr error
	provider, providerErr = acmedns.NewDNSProvider()
	if providerErr != nil {
		return fmt.Errorf("failed to create acme-dns provider: %w", providerErr)
	}

	// Set up the DNS-01 provider with proper resolver configuration
	var dnsErr error
	if cfg.DnsResolver != "" {
		// Format nameserver address correctly (add :53 if port is missing)
		nsAddr := cfg.DnsResolver
		if !strings.Contains(nsAddr, ":") {
			nsAddr += ":53"
		}

		// Create a slice of nameservers with the custom resolver
		nameservers := []string{nsAddr}
		DefaultLogger.Infof("Configuring DNS-01 challenge with custom nameservers: %v", nameservers)

		// Set DNS01 provider with custom recursive nameservers
		dnsErr = client.Challenge.SetDNS01Provider(
			provider,
			dns01.AddRecursiveNameservers(nameservers),
			dns01.DisableCompletePropagationRequirement(),
		)
	} else {
		// Default case - use the provider as is
		dnsErr = client.Challenge.SetDNS01Provider(provider)
	}

	if dnsErr != nil {
		return fmt.Errorf("failed to set DNS01 provider: %w", dnsErr)
	}

	// Register the user if needed
	if user.Registration == nil {
		DefaultLogger.Info("No existing ACME registration found. Registering...")
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return fmt.Errorf("ACME registration failed: %w", err)
		}
		user.Registration = reg
		DefaultLogger.Info("ACME registration successful.")
		if err := saveUser(cfg, user); err != nil {
			// Log error but continue, registration succeeded
			DefaultLogger.Warnf("Warning: failed to save ACME registration details: %v", err)
		}
	} else {
		DefaultLogger.Info("Using existing ACME registration.")
	}

	// Perform the requested action
	switch action {
	case "init":
		DefaultLogger.Infof("Requesting new certificate for domains: %v", domainsToProcess)

		// ACME-DNS setup was already verified in PreCheckAcmeDNS, so we can proceed directly
		request := certificate.ObtainRequest{
			Domains: domainsToProcess, // Use domainsToProcess
			Bundle:  true,             // Get certificate chain
		}
		certificates, err := client.Certificate.Obtain(request)
		if err != nil {
			return fmt.Errorf("failed to obtain certificate: %w", err)
		}
		DefaultLogger.Infof("Successfully obtained certificate '%s'!", certName)
		// Lego automatically saves certs based on its internal storage logic,
		// which relies on the working directory or can be configured.
		// We need to ensure it saves to cfg.LegoStoragePath/certificates
		// Pass certName to saveCertificates
		if err := saveCertificates(cfg, certName, certificates); err != nil {
			DefaultLogger.Warnf("Warning: failed to save certificate '%s': %v", certName, err)
		}
	case "renew":
		// Renewal typically renews the *existing* certificate identified by its primary domain,
		// which should cover all domains listed in the cert. Lego's Renew function handles this.
		// We just need the primary domain from the list to load the existing cert resource.
		primaryDomain := domainsToProcess[0]
		DefaultLogger.Infof("Attempting to renew certificate associated with primary domain %s (covers: %v)", primaryDomain, domainsToProcess)

		// Check if the certificate resource file exists for the certificate name.
		metaPath := filepath.Join(cfg.CertStoragePath, "certificates", fmt.Sprintf("%s.json", certName)) // Use renamed field
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			// Also check the .crt file for robustness
			certPath := filepath.Join(cfg.CertStoragePath, "certificates", fmt.Sprintf("%s.crt", certName)) // Use renamed field
			if _, err := os.Stat(certPath); os.IsNotExist(err) {
				return fmt.Errorf("cannot renew: certificate file not found for certificate %s at %s (and %s). Run 'init' first?", certName, certPath, metaPath)
			}
			DefaultLogger.Warnf("Warning: Certificate metadata file %s missing, but certificate %s exists. Attempting renewal but might lack SANs.", metaPath, certPath)
			// Proceed without existingCert, Lego might handle it? Or fail.
			// Let's require the metadata for reliable renewal.
			return fmt.Errorf("cannot renew: certificate metadata file not found at %s. Run 'init' again?", metaPath)

		} else if err != nil {
			return fmt.Errorf("cannot renew: error checking certificate metadata file %s: %w", metaPath, err)
		}

		// Lego's Renew function handles loading internally if paths are standard,
		// but let's be explicit or ensure the internal storage points correctly.
		// For now, assume Lego handles loading based on domain if storage is consistent.
		// A more robust approach might load the cert resource manually.

		renewOptions := certificate.RenewOptions{
			// Days: 30, // Renew if expiring within 30 days (Lego default)
			Bundle: true,
		}

		// Note: Lego's Renew function expects the *certificate resource*.
		// We load it using the certificate name.
		existingCert, err := LoadCertificateResource(cfg, certName) // Use certName and exported func
		if err != nil {
			return fmt.Errorf("failed to load existing certificate resource for '%s' for renewal: %w", certName, err)
		}

		newCertificates, err := client.Certificate.Renew(*existingCert, renewOptions.Bundle, renewOptions.MustStaple, renewOptions.PreferredChain)
		if err != nil {
			return fmt.Errorf("failed to renew certificate: %w", err)
		}

		// Check if renewal actually occurred (Lego might return the old cert if still valid)
		if newCertificates == nil || string(newCertificates.Certificate) == string(existingCert.Certificate) {
			DefaultLogger.Info("Certificate renewal not required or did not result in a new certificate.")
		} else {
			DefaultLogger.Infof("Successfully renewed certificate '%s'!", certName)
			// Pass certName to saveCertificates
			if err := saveCertificates(cfg, certName, newCertificates); err != nil {
				DefaultLogger.Warnf("Warning: failed to save renewed certificate '%s': %v", certName, err)
			}
		}
	default:
		// Handle unknown action
		return fmt.Errorf("internal error: unsupported action '%s'", action)
	}

	return nil
}
