//go:build testutils
// +build testutils

package test_helpers

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// CertificateInfo contains metadata about a certificate
type CertificateInfo struct {
	CommonName string
	DNSNames   []string
	NotBefore  time.Time
	NotAfter   time.Time
	Issuer     string
}

// ValidateCertificateFile reads a certificate file and validates its contents
func ValidateCertificateFile(t *testing.T, certPath string) (*CertificateInfo, error) {
	// Read the certificate file
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	// Parse the PEM data
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("failed to parse certificate PEM")
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	// Extract certificate information
	info := &CertificateInfo{
		CommonName: cert.Subject.CommonName,
		DNSNames:   cert.DNSNames,
		NotBefore:  cert.NotBefore,
		NotAfter:   cert.NotAfter,
		Issuer:     cert.Issuer.CommonName,
	}

	return info, nil
}

// ValidateCertificateExpiry checks if a certificate is within its validity period
func ValidateCertificateExpiry(t *testing.T, certPath string) (bool, error) {
	info, err := ValidateCertificateFile(t, certPath)
	if err != nil {
		return false, err
	}

	now := time.Now()
	isValid := now.After(info.NotBefore) && now.Before(info.NotAfter)

	if !isValid {
		t.Logf("Certificate %s expired or not yet valid", certPath)
		t.Logf("   Valid from: %s", info.NotBefore)
		t.Logf("   Valid to: %s", info.NotAfter)
		t.Logf("   Current time: %s", now)
	}

	return isValid, nil
}

// ValidateCertificateDomains checks if a certificate covers the expected domains
func ValidateCertificateDomains(t *testing.T, certPath string, expectedDomains []string) (bool, error) {
	info, err := ValidateCertificateFile(t, certPath)
	if err != nil {
		return false, err
	}

	// Check if all expected domains are present in the certificate
	missingDomains := []string{}
	domainMap := make(map[string]bool)

	for _, domain := range info.DNSNames {
		domainMap[domain] = true
	}

	// Add CommonName to the domain map if it's not already in DNSNames
	if info.CommonName != "" && !domainMap[info.CommonName] {
		domainMap[info.CommonName] = true
	}

	for _, expected := range expectedDomains {
		if !domainMap[expected] {
			missingDomains = append(missingDomains, expected)
		}
	}

	if len(missingDomains) > 0 {
		t.Logf("Certificate %s is missing domains: %v", certPath, missingDomains)
		t.Logf("   Certificate domains: %v", info.DNSNames)
		return false, nil
	}

	return true, nil
}

// ValidateAllCertificateFiles checks all certificate files in a directory
func ValidateAllCertificateFiles(t *testing.T, certsDir string) (int, int, error) {
	// Find all .crt files in the directory
	certFiles, err := filepath.Glob(filepath.Join(certsDir, "*.crt"))
	if err != nil {
		return 0, 0, err
	}

	var validCount, invalidCount int

	for _, certFile := range certFiles {
		// Skip issuer certificates
		if filepath.Base(certFile) == "issuer.crt" ||
			filepath.Base(certFile) == "ca.crt" ||
			filepath.Base(certFile) == "chain.crt" {
			continue
		}

		// Validate expiry
		isValid, err := ValidateCertificateExpiry(t, certFile)
		if err != nil {
			t.Logf("Error validating %s: %v", certFile, err)
			invalidCount++
			continue
		}

		if isValid {
			validCount++
		} else {
			invalidCount++
		}
	}

	return validCount, invalidCount, nil
}
