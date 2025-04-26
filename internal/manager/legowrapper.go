package manager // Changed from main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"os" // Added for Setenv
	"path/filepath"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/acmedns"
	"github.com/go-acme/lego/v4/registration"
)

// Simple ACME User struct implementing registration.User
type MyUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *MyUser) GetEmail() string {
	return u.Email
}
func (u *MyUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *MyUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// createOrLoadUser creates a new ACME user or loads an existing one from storage.
func createOrLoadUser(cfg *Config) (*MyUser, error) {
	// Determine storage path relative to config file
	storageDir := cfg.CertStoragePath // Use renamed field
	if err := os.MkdirAll(storageDir, DirPermissions); err != nil {
		return nil, fmt.Errorf("creating cert storage directory %s: %w", storageDir, err)
	}

	accountFilePath := filepath.Join(storageDir, "account.json")
	keyFilePath := filepath.Join(storageDir, "account.key")

	var privateKey crypto.PrivateKey
	var err error

	// Check if key file exists
	if _, err = os.Stat(keyFilePath); os.IsNotExist(err) {
		log.Printf("Generating new private key (%s) for ACME account", cfg.KeyType)
		// Generate new key based on config
		switch cfg.KeyType {
		case "rsa2048":
			privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
		case "rsa4096":
			privateKey, err = rsa.GenerateKey(rand.Reader, 4096)
		case "rsa8192":
			privateKey, err = rsa.GenerateKey(rand.Reader, 8192)
		case "ec256":
			privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		case "ec384":
			privateKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		default:
			return nil, fmt.Errorf("unsupported key type: %s", cfg.KeyType)
		}
		if err != nil {
			return nil, fmt.Errorf("generating private key: %w", err)
		}
		// Save the new key
		// Corrected assignment: PEMEncode returns only one value
		keyBytes := certcrypto.PEMEncode(privateKey)
		// Removed the err check here as PEMEncode doesn't return an error directly in this context
		// if err != nil {
		// 	return nil, fmt.Errorf("encoding private key: %w", err)
		// }
		if err = os.WriteFile(keyFilePath, keyBytes, PrivateKeyPermissions); err != nil {
			return nil, fmt.Errorf("saving private key to %s: %w", keyFilePath, err)
		}
		log.Printf("Saved new private key to %s", keyFilePath)
	} else if err != nil {
		return nil, fmt.Errorf("checking private key file %s: %w", keyFilePath, err)
	} else {
		// Load existing key
		log.Printf("Loading existing private key from %s", keyFilePath)
		keyBytes, err := os.ReadFile(keyFilePath)
		if err != nil {
			return nil, fmt.Errorf("reading private key file %s: %w", keyFilePath, err)
		}
		privateKey, err = certcrypto.ParsePEMPrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing private key from %s: %w", keyFilePath, err)
		}
	}

	user := &MyUser{
		Email: cfg.Email,
		key:   privateKey,
	}

	// Load registration info if it exists
	if _, err := os.Stat(accountFilePath); err == nil {
		log.Printf("Loading existing ACME registration from %s", accountFilePath)
		accountBytes, err := os.ReadFile(accountFilePath)
		if err != nil {
			return nil, fmt.Errorf("reading account file %s: %w", accountFilePath, err)
		}
		err = json.Unmarshal(accountBytes, &user.Registration)
		if err != nil {
			return nil, fmt.Errorf("parsing account file %s: %w", accountFilePath, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("checking account file %s: %w", accountFilePath, err)
	}

	return user, nil
}

// saveUser saves the user's registration resource.
func saveUser(cfg *Config, user *MyUser) error {
	if user.Registration == nil {
		return fmt.Errorf("cannot save user without registration resource")
	}
	storageDir := cfg.CertStoragePath // Use renamed field
	accountFilePath := filepath.Join(storageDir, "account.json")

	regBytes, err := json.MarshalIndent(user.Registration, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling registration resource: %w", err)
	}

	err = os.WriteFile(accountFilePath, regBytes, PrivateKeyPermissions)
	if err != nil {
		return fmt.Errorf("writing account file %s: %w", accountFilePath, err)
	}
	log.Printf("Saved ACME registration to %s", accountFilePath)
	return nil
}

// RunLego performs the certificate obtain or renew operation.
// Accepts config, account store, action, the certificate name, and the domains list.
// Exported function
func RunLego(cfg *Config, store *accountStore, action string, certName string, domainsToProcess []string) error {
	log.Println("Initializing Lego client...")

	// Validate domainsToProcess is not empty (should be caught by main, but good practice)
	if len(domainsToProcess) == 0 {
		return fmt.Errorf("RunLego called with empty domains list")
	}

	user, err := createOrLoadUser(cfg)
	if err != nil {
		return fmt.Errorf("failed to create/load ACME user: %w", err)
	}

	// Setup Lego config
	legoConfig := lego.NewConfig(user)
	legoConfig.CADirURL = cfg.AcmeServer                             // Use renamed field
	legoConfig.Certificate.KeyType = certcrypto.KeyType(cfg.KeyType) // Use configured key type
	legoConfig.Certificate.Timeout = 30 * time.Minute                // Generous timeout for challenges

	// Create Lego client
	client, err := lego.NewClient(legoConfig)
	if err != nil {
		return fmt.Errorf("failed to create Lego client: %w", err)
	}

	// Setup acme-dns provider
	// The provider reads ACME_DNS_API_BASE and ACME_DNS_STORAGE_PATH from env vars.
	// We set them explicitly here from our config to avoid implicit dependencies.
	log.Printf("Setting ACME_DNS_API_BASE=%s", cfg.AcmeDnsServer)
	if err := os.Setenv("ACME_DNS_API_BASE", cfg.AcmeDnsServer); err != nil {
		return fmt.Errorf("failed to set ACME_DNS_API_BASE env var: %w", err)
	}
	// The acmedns provider uses the storage path to *read* the credentials from the JSON file.
	log.Printf("Setting ACME_DNS_STORAGE_PATH=%s", store.filePath) // Use store.filePath
	if err := os.Setenv("ACME_DNS_STORAGE_PATH", store.filePath); err != nil {
		return fmt.Errorf("failed to set ACME_DNS_STORAGE_PATH env var: %w", err)
	}

	// Create the DNS provider using the environment variables we've set
	provider, err := acmedns.NewDNSProvider()
	if err != nil {
		return fmt.Errorf("failed to create acme-dns provider: %w", err)
	}

	err = client.Challenge.SetDNS01Provider(provider)
	if err != nil {
		return fmt.Errorf("failed to set DNS01 provider: %w", err)
	}

	// Register the user if needed
	if user.Registration == nil {
		log.Println("No existing ACME registration found. Registering...")
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return fmt.Errorf("ACME registration failed: %w", err)
		}
		user.Registration = reg
		log.Println("ACME registration successful.")
		if err := saveUser(cfg, user); err != nil {
			// Log error but continue, registration succeeded
			log.Printf("Warning: failed to save ACME registration details: %v", err)
		}
	} else {
		log.Println("Using existing ACME registration.")
	}

	// Perform the requested action
	if action == "init" {
		log.Printf("Requesting new certificate for domains: %v", domainsToProcess) // Use domainsToProcess
		request := certificate.ObtainRequest{
			Domains: domainsToProcess, // Use domainsToProcess
			Bundle:  true,             // Get certificate chain
		}
		certificates, err := client.Certificate.Obtain(request)
		if err != nil {
			return fmt.Errorf("failed to obtain certificate: %w", err)
		}
		log.Printf("Successfully obtained certificate '%s'!", certName)
		// Lego automatically saves certs based on its internal storage logic,
		// which relies on the working directory or can be configured.
		// We need to ensure it saves to cfg.LegoStoragePath/certificates
		// Pass certName to saveCertificates
		if err := saveCertificates(cfg, certName, certificates); err != nil {
			log.Printf("Warning: failed to save certificate '%s': %v", certName, err)
		}
	} else if action == "renew" {
		// Renewal typically renews the *existing* certificate identified by its primary domain,
		// which should cover all domains listed in the cert. Lego's Renew function handles this.
		// We just need the primary domain from the list to load the existing cert resource.
		primaryDomain := domainsToProcess[0]
		log.Printf("Attempting to renew certificate associated with primary domain %s (covers: %v)", primaryDomain, domainsToProcess)

		// Check if the certificate resource file exists for the primary domain.
		metaPath := filepath.Join(cfg.CertStoragePath, "certificates", fmt.Sprintf("%s.json", primaryDomain)) // Use renamed field
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			// Also check the .crt file for robustness
			certPath := filepath.Join(cfg.CertStoragePath, "certificates", fmt.Sprintf("%s.crt", primaryDomain)) // Use renamed field
			if _, err := os.Stat(certPath); os.IsNotExist(err) {
				return fmt.Errorf("cannot renew: certificate file not found for primary domain %s at %s (and %s). Run 'init' first?", primaryDomain, certPath, metaPath)
			}
			log.Printf("Warning: Certificate metadata file %s missing, but certificate %s exists. Attempting renewal but might lack SANs.", metaPath, certPath)
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
			log.Println("Certificate renewal not required or did not result in a new certificate.")
		} else {
			log.Printf("Successfully renewed certificate '%s'!", certName)
			// Pass certName to saveCertificates
			if err := saveCertificates(cfg, certName, newCertificates); err != nil {
				log.Printf("Warning: failed to save renewed certificate '%s': %v", certName, err)
			}
		}
	} else {
		return fmt.Errorf("internal error: unsupported action '%s'", action)
	}

	return nil
}

// saveCertificates saves the obtained certificate files using the certName.
func saveCertificates(cfg *Config, certName string, resource *certificate.Resource) error {
	certsDir := filepath.Join(cfg.CertStoragePath, "certificates") // Use renamed field
	if err := os.MkdirAll(certsDir, DirPermissions); err != nil {
		return fmt.Errorf("creating certificates directory %s: %w", certsDir, err)
	}

	// Use the provided certName for filenames
	certFile := filepath.Join(certsDir, certName+".crt")
	keyFile := filepath.Join(certsDir, certName+".key")
	issuerFile := filepath.Join(certsDir, certName+".issuer.crt")
	jsonFile := filepath.Join(certsDir, certName+".json")

	// Ensure resource.Domain is set correctly, use certName if primary domain isn't obvious
	// Lego usually sets resource.Domain to the first domain in the request.
	if resource.Domain == "" {
		log.Printf("Warning: certificate.Resource.Domain is empty, using certName '%s' for metadata.", certName)
		resource.Domain = certName // Or maybe the first domain from the request? Let's stick to certName for consistency.
	}

	err := os.WriteFile(certFile, resource.Certificate, CertificatePermissions)
	if err != nil {
		return fmt.Errorf("writing certificate file %s: %w", certFile, err)
	}
	log.Printf("Saved certificate to %s", certFile)

	err = os.WriteFile(keyFile, resource.PrivateKey, PrivateKeyPermissions)
	if err != nil {
		return fmt.Errorf("writing private key file %s: %w", keyFile, err)
	}
	log.Printf("Saved private key to %s", keyFile)

	// Save issuer certificate if present
	if len(resource.IssuerCertificate) > 0 {
		err = os.WriteFile(issuerFile, resource.IssuerCertificate, CertificatePermissions)
		if err != nil {
			// Non-fatal, just log
			log.Printf("Warning: writing issuer certificate file %s: %v", issuerFile, err)
		} else {
			log.Printf("Saved issuer certificate to %s", issuerFile)
		}
	}

	// Save metadata
	jsonBytes, err := json.MarshalIndent(resource, "", "  ")
	if err != nil {
		// Use certName in the error message
		return fmt.Errorf("marshalling certificate metadata for %s: %w", certName, err)
	}
	err = os.WriteFile(jsonFile, jsonBytes, PrivateKeyPermissions)
	if err != nil {
		return fmt.Errorf("writing certificate metadata file %s: %w", jsonFile, err)
	}
	log.Printf("Saved certificate metadata to %s", jsonFile)

	return nil
}

// LoadCertificateResource loads the certificate metadata from the JSON file.
// Exported function. Accepts certName instead of domain.
func LoadCertificateResource(cfg *Config, certName string) (*certificate.Resource, error) {
	jsonFile := filepath.Join(cfg.CertStoragePath, "certificates", fmt.Sprintf("%s.json", certName)) // Use renamed field

	if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
		// It's okay if the file doesn't exist (e.g., for 'init' action), return specific error?
		// Or let the caller handle os.IsNotExist. Let's return the error.
		return nil, err // Return the os.IsNotExist error
	} else if err != nil {
		// Other stat error
		return nil, fmt.Errorf("checking certificate metadata file %s: %w", jsonFile, err)
	}

	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("reading certificate metadata file %s: %w", jsonFile, err)
	}

	var resource certificate.Resource
	err = json.Unmarshal(data, &resource)
	if err != nil {
		return nil, fmt.Errorf("parsing certificate metadata file %s: %w", jsonFile, err)
	}

	// We also need to load the private key associated with the certificate
	keyFile := filepath.Join(cfg.CertStoragePath, "certificates", fmt.Sprintf("%s.key", certName)) // Use renamed field
	keyBytes, err := os.ReadFile(keyFile)
	if err != nil {
		// If the key is missing, that's a problem for renewal
		return nil, fmt.Errorf("reading certificate private key file %s: %w", keyFile, err)
	}
	resource.PrivateKey = keyBytes // Lego expects the raw bytes here for renewal

	// Load the actual certificate file content too
	certFile := filepath.Join(cfg.CertStoragePath, "certificates", fmt.Sprintf("%s.crt", certName)) // Use renamed field
	certBytes, err := os.ReadFile(certFile)
	if err != nil {
		// If the cert file is missing, also a problem
		return nil, fmt.Errorf("reading certificate file %s: %w", certFile, err)
	}
	resource.Certificate = certBytes

	return &resource, nil
}
