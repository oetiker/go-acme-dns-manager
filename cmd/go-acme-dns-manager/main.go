package main

import (
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath" // For comparing domain lists

	// "sort" // Removed unused import
	"strings"
	"time"

	"github.com/oetiker/go-acme-dns-manager/internal/manager"
)

// Define a struct to hold parsed certificate requests
type certRequest struct {
	Name    string
	Domains []string
	KeyType string
}

var (
	configPath          = flag.String("config", "config.yaml", "Path to the configuration file")
	autoMode            = flag.Bool("auto", false, "Enable automatic mode using 'auto_domains' config section (handles init and renew)")
	printConfigTemplate = flag.Bool("print-config-template", false, "Print a default configuration template to stdout and exit")
	migrateAccounts     = flag.Bool("migrate-accounts", false, "Migrate accounts from old format (account.json/account.key) to new server-specific directory structure")
)

// Helper function to parse cert-name@domain1,domain2/key_type=ec384 syntax
// This function is used for testing purposes only now
func parseCertArg(arg string) (string, []string, string, error) {
	// Check for key_type parameter
	keyType := ""
	domainPart := arg

	if strings.Contains(arg, "/") {
		argParts := strings.Split(arg, "/")
		domainPart = argParts[0]

		// Process any parameters after the slash
		for i := 1; i < len(argParts); i++ {
			param := argParts[i]
			if strings.HasPrefix(param, "key_type=") {
				keyType = strings.TrimPrefix(param, "key_type=")
				log.Printf("Found key_type parameter: %s", keyType)
			} else {
				log.Printf("Warning: Unknown parameter in certificate spec: %s", param)
				// This doesn't make the command fail, just a warning
			}
		}
	}

	// Process the domain part
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
			domains = append(domains, trimmed)
		}
	}
	if len(domains) == 0 {
		return "", nil, "", fmt.Errorf("no valid domains found after '@' in argument '%s'", domainPart)
	}
	// Basic validation for cert name (adjust regex as needed for stricter rules)
	// For now, just check it's not empty and doesn't contain problematic chars like '/' or '\'
	if strings.ContainsAny(certName, "/\\") {
		return "", nil, "", fmt.Errorf("invalid certificate name '%s': must not contain '/' or '\\'", certName)
	}

	return certName, domains, keyType, nil
}

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

	// Handle print config template flag first
	if *printConfigTemplate {
		fmt.Println("# Default configuration template:")
		err := manager.GenerateDefaultConfig(os.Stdout) // Write to stdout
		if err != nil {
			log.Fatalf("Error printing config template: %v", err)
		}
		os.Exit(0)
	}

	// --- Config Loading ---
	absConfigPath, err := filepath.Abs(*configPath)
	if err != nil {
		log.Fatalf("Error getting absolute path for config file %s: %v", *configPath, err)
	}
	*configPath = absConfigPath

	// Check if config file exists, error if not (don't generate automatically)
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		log.Fatalf("Error: Configuration file not found at %s. Use -print-config-template to get a template.", *configPath)
	} else if err != nil {
		log.Fatalf("Error checking config file %s: %v", *configPath, err)
	}

	// Load configuration (file must exist at this point)
	fmt.Printf("Loading configuration from %s...\n", *configPath)
	cfg, err := manager.LoadConfig(*configPath)
	if err != nil {
		// Check for placeholder email only, as domains list is removed/optional
		contentBytes, readErr := os.ReadFile(*configPath)
		if readErr == nil {
			content := string(contentBytes)
			if strings.Contains(content, "your-email@example.com") {
				log.Fatalf("Error: Configuration file %s still contains placeholder email. Please edit it.", *configPath)
			}
		}
		log.Fatalf("Error loading config file %s: %v", *configPath, err)
	}
	fmt.Println("Configuration loaded successfully.")

	// --- Mode Determination ---
	positionalArgs := flag.Args()
	isManualMode := len(positionalArgs) > 0
	isAutoMode := *autoMode // Use renamed flag variable

	if isManualMode && isAutoMode {
		log.Fatal("Error: Cannot use -auto flag and specify certificate arguments simultaneously.")
	}
	if !isManualMode && !isAutoMode && !*migrateAccounts {
		fmt.Fprintf(os.Stderr, "Error: No operation specified. Provide certificate arguments or use -auto flag.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// --- Account Store Initialization ---
	accountsFilePath := filepath.Join(cfg.CertStoragePath, "acme-dns-accounts.json") // Use renamed field
	fmt.Printf("Loading ACME DNS accounts from %s...\n", accountsFilePath)
	store, err := manager.NewAccountStore(accountsFilePath)
	if err != nil {
		log.Fatalf("Error initializing account store from %s: %v", accountsFilePath, err)
	}
	fmt.Println("ACME DNS accounts loaded successfully.")

	// --- Build List of Certificate Requests ---
	requests := []certRequest{}
	requestedNames := make(map[string]struct{}) // For duplicate name check

	if isManualMode {
		log.Println("Mode: Manual Specification")
		for _, arg := range positionalArgs {
			var certName string
			var domains []string
			var keyType string
			// First check if the arg contains key_type parameter
			var domainPart string
			if strings.Contains(arg, "/") {
				parts := strings.Split(arg, "/")
				domainPart = parts[0]

				// Extract key_type if present
				for i := 1; i < len(parts); i++ {
					param := parts[i]
					if strings.HasPrefix(param, "key_type=") {
						keyType = strings.TrimPrefix(param, "key_type=")
						log.Printf("Found key_type parameter: %s", keyType)
					} else {
						log.Printf("Warning: Unknown parameter in certificate spec: %s", param)
					}
				}
			} else {
				domainPart = arg
			}

			if strings.Contains(domainPart, "@") {
				// Use explicit format: cert-name@domain1,domain2,...
				parts := strings.SplitN(domainPart, "@", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					log.Fatalf("Error: Invalid certificate format '%s'. Expected 'name@domain.com'", domainPart)
				}
				certName = parts[0]
				domains = []string{}
				rawDomains := strings.Split(parts[1], ",")
				for _, d := range rawDomains {
					trimmed := strings.TrimSpace(d)
					if trimmed != "" {
						domains = append(domains, trimmed)
					}
				}
			} else {
				// Use shorthand: treat arg as both cert name and single domain
				certName = domainPart
				domains = []string{domainPart}
				log.Printf("Interpreting argument '%s' as shorthand for '%s@%s'", domainPart, certName, certName)
				// Basic validation for shorthand name
				if strings.ContainsAny(certName, "/\\") {
					log.Fatalf("invalid certificate name '%s': must not contain '/' or '\\'", certName)
				}
				if certName == "" {
					log.Fatalf("Error: Empty certificate name/domain provided.")
				}
			}

			if _, exists := requestedNames[certName]; exists {
				log.Fatalf("Error: Duplicate certificate name specified or implied in arguments: '%s'", certName)
			}
			requests = append(requests, certRequest{Name: certName, Domains: domains, KeyType: keyType})
			requestedNames[certName] = struct{}{}
		}
	} else { // Auto Mode
		log.Println("Mode: Automatic") // Update log message
		if cfg.AutoDomains == nil || len(cfg.AutoDomains.Certs) == 0 {
			log.Println("No certificates defined in 'auto_domains.certs' section of the config file. Nothing to do.")
			os.Exit(0)
		}
		log.Printf("Processing %d certificate definition(s) from config file...", len(cfg.AutoDomains.Certs))
		for name, certDef := range cfg.AutoDomains.Certs {
			// Basic validation already done in LoadConfig
			requests = append(requests, certRequest{Name: name, Domains: certDef.Domains, KeyType: certDef.KeyType})
			if certDef.KeyType != "" {
				log.Printf("Certificate %s will use key type: %s", name, certDef.KeyType)
			}
			// No need to check for duplicate names here as map keys are unique
		}
	}

	// --- Pre-Check for Conflicts and Determine Actions ---
	log.Println("Performing pre-checks for requested certificates...")
	type requestTask struct {
		Request certRequest
		Action  string // "init", "renew", "skip"
	}
	tasks := []requestTask{}
	renewalThreshold := cfg.GetRenewalThreshold() // Get duration from config/default

	for _, req := range requests {
		log.Printf("Checking certificate: %s (%v)", req.Name, req.Domains)
		action := "init" // Default action is init
		// skip := false // Removed unused variable

		metaPath := filepath.Join(cfg.CertStoragePath, "certificates", req.Name+".json") // Use renamed field
		certPath := filepath.Join(cfg.CertStoragePath, "certificates", req.Name+".crt")  // Use renamed field

		if _, err := os.Stat(metaPath); err == nil {
			// Metadata exists, potential renew
			action = "renew"
			log.Printf("  Existing metadata found (%s). Checking domains and expiry.", metaPath)

			// Load existing metadata to check domains
			existingCertData, err := manager.LoadCertificateResource(cfg, req.Name) // Assuming LoadCertificateResource exists
			if err != nil {
				log.Fatalf("Error loading existing certificate metadata for '%s' from %s: %v", req.Name, metaPath, err)
			}

			// Simplified Check: Compare only the primary domain.
			// A full SAN list comparison seems problematic with the loaded resource struct.
			// This ensures the main domain matches the existing cert.
			if len(req.Domains) > 0 && req.Domains[0] != existingCertData.Domain {
				log.Fatalf("Error: Primary domain mismatch for certificate '%s'.\n  Requested primary: %s\n  Existing primary (%s): %s\nPlease use a different certificate name or manually remove the old files.",
					req.Name, req.Domains[0], metaPath, existingCertData.Domain)
			} else if len(req.Domains) == 0 {
				// Should not happen due to earlier parsing checks, but safety first
				log.Fatalf("Internal Error: Empty domain list for certificate request '%s'", req.Name)
			} else {
				log.Printf("  Primary domain '%s' matches existing certificate.", req.Domains[0])

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
								log.Printf("  Warning: Domain list differences detected for certificate '%s':", req.Name)
								if len(missingDomains) > 0 {
									log.Printf("    - New domains that will be added: %v", missingDomains)
								}
								if len(extraDomains) > 0 {
									log.Printf("    - Domains in existing cert that will be removed: %v", extraDomains)
								}
								log.Printf("  If this is not intended, please use a different certificate name.")
							} else {
								log.Printf("  All domains match between requested domains and existing certificate.")
							}
						} else {
							log.Printf("  Warning: Could not parse certificate from %s: %v. Skipping SAN comparison.", certPath, err)
						}
					} else {
						log.Printf("  Warning: Failed to decode PEM block from %s. Skipping SAN comparison.", certPath)
					}
				} else {
					log.Printf("  Warning: Could not read certificate file %s: %v. Skipping SAN comparison.", certPath, err)
				}

			}

			// If in auto mode, check expiry date
			if isAutoMode { // Use renamed flag variable
				certBytes, err := os.ReadFile(certPath)
				if err != nil {
					log.Printf("  Warning: Could not read existing certificate file %s for expiry check: %v. Proceeding with renewal.", certPath, err)
				} else {
					block, _ := pem.Decode(certBytes)
					if block == nil {
						log.Printf("  Warning: Failed to decode PEM block from %s. Proceeding with renewal.", certPath)
					} else {
						cert, err := x509.ParseCertificate(block.Bytes)
						if err != nil {
							log.Printf("  Warning: Failed to parse certificate from %s: %v. Proceeding with renewal.", certPath, err)
						} else {
							timeLeft := time.Until(cert.NotAfter)
							log.Printf("  Certificate expires on %s (%v remaining). Renewal threshold is %v.", cert.NotAfter.Format(time.RFC1123), timeLeft.Round(time.Hour), renewalThreshold)
							if timeLeft > renewalThreshold {
								log.Printf("  Skipping renewal: Certificate is not within the renewal threshold.")
								action = "skip" // Mark as skip
							} else {
								log.Printf("  Certificate is within renewal threshold. Proceeding with renewal.")
							}
						}
					}
				}
			}

		} else if !os.IsNotExist(err) {
			// Error checking file other than not found
			log.Fatalf("Error checking certificate metadata file %s: %v", metaPath, err)
		} else {
			log.Printf("  No existing metadata found (%s). Action set to 'init'.", metaPath)
		}

		if action != "skip" {
			tasks = append(tasks, requestTask{Request: req, Action: action})
		}
	} // End pre-check loop

	if len(tasks) == 0 {
		log.Println("No certificates require processing.")
		os.Exit(0)
	}

	log.Printf("Pre-checks complete. Processing %d certificate task(s)...", len(tasks))

	// --- Process Tasks (ACME DNS Verification & Lego Execution) ---
	anyFailure := false
	for _, task := range tasks {
		certName := task.Request.Name
		domains := task.Request.Domains
		action := task.Action

		log.Printf("--- Processing Task: Action=%s, CertName=%s, Domains=%v ---", action, certName, domains)

		// 1. Verify/Register ACME DNS for all domains in this group
		needsManualUpdate := false
		log.Printf("Verifying/Registering ACME DNS accounts for %d domain(s)...", len(domains))
		for _, domain := range domains {
			account, exists := store.GetAccount(domain)

			if !exists {
				log.Printf("  Registering ACME DNS for %s...", domain)
				newAccount, err := manager.RegisterNewAccount(cfg, store, domain)
				if err != nil {
					log.Printf("  ERROR registering new acme-dns account for %s: %v", domain, err)
					anyFailure = true
					break // Stop processing this cert group if registration fails
				}
				manager.PrintRequiredCname(domain, newAccount.FullDomain)
				needsManualUpdate = true
				continue
			}

			// Account exists, verify CNAME
			log.Printf("  Verifying CNAME for %s...", domain)
			cnameValid, err := manager.VerifyCnameRecord(cfg, domain, account.FullDomain)
			if err != nil {
				log.Printf("  Warning: Error verifying CNAME record for %s: %v. Treating as invalid.", domain, err)
				manager.PrintRequiredCname(domain, account.FullDomain)
				needsManualUpdate = true
			} else if !cnameValid {
				log.Printf("  CNAME record for %s is missing or invalid.", domain)
				manager.PrintRequiredCname(domain, account.FullDomain)
				needsManualUpdate = true
			} else {
				log.Printf("  CNAME record for %s is valid.", domain)
			}
		} // End domain loop for ACME DNS

		if anyFailure {
			log.Printf("Skipping Lego operation for '%s' due to previous errors.", certName)
			continue // Move to the next task
		}

		if needsManualUpdate {
			log.Printf("Manual DNS CNAME updates required for certificate '%s'. Please configure the records shown above and run again.", certName)
			anyFailure = true // Treat as failure for overall exit code
			continue          // Move to the next task
		}

		log.Printf("All domains for '%s' have valid ACME DNS configurations.", certName)

		// 2. Run Lego action
		log.Printf("Proceeding with Lego action '%s' for certificate '%s'...", action, certName)
		keyType := task.Request.KeyType
		if keyType != "" {
			log.Printf("Using specified key type for certificate: %s", keyType)
		}
		err = manager.RunLego(cfg, store, action, certName, domains, keyType) // Pass certName and keyType
		if err != nil {
			log.Printf("ERROR: Lego operation failed for certificate '%s': %v", certName, err)
			anyFailure = true
			// Continue to next task even if one fails? Or stop? Let's continue for now.
		} else {
			log.Printf("Lego operation successful for certificate '%s'.", certName)
		}

	} // End task processing loop

	// --- Final Status ---
	if anyFailure {
		log.Println("\nOne or more operations failed or require manual intervention.")
		os.Exit(1)
	}

	log.Println("\nOperation completed successfully.")
}
