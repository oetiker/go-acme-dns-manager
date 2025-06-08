package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-acme/lego/v4/certificate"
)

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
		DefaultLogger.Warnf("Warning: certificate.Resource.Domain is empty, using certName '%s' for metadata.", certName)
		resource.Domain = certName // Or maybe the first domain from the request? Let's stick to certName for consistency.
	}

	err := os.WriteFile(certFile, resource.Certificate, CertificatePermissions)
	if err != nil {
		return fmt.Errorf("writing certificate file %s: %w", certFile, err)
	}
	DefaultLogger.Infof("Saved certificate to %s", certFile)

	err = os.WriteFile(keyFile, resource.PrivateKey, PrivateKeyPermissions)
	if err != nil {
		return fmt.Errorf("writing private key file %s: %w", keyFile, err)
	}
	DefaultLogger.Infof("Saved private key to %s", keyFile)

	// Save issuer certificate if present
	if len(resource.IssuerCertificate) > 0 {
		err = os.WriteFile(issuerFile, resource.IssuerCertificate, CertificatePermissions)
		if err != nil {
			// Non-fatal, just log
			DefaultLogger.Warnf("Warning: writing issuer certificate file %s: %v", issuerFile, err)
		} else {
			DefaultLogger.Infof("Saved issuer certificate to %s", issuerFile)
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
	DefaultLogger.Infof("Saved certificate metadata to %s", jsonFile)

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
