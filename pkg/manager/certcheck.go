// Package manager provides functions for managing ACME DNS certificates.
package manager

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// CertificateNeedsRenewal checks if a certificate needs renewal based on:
// 1. Expiry time (if it expires within renewalThreshold)
// 2. Domain changes (if requested domains are not all in the certificate)
// Returns whether renewal is needed, reason for renewal, and any error encountered
func CertificateNeedsRenewal(certPath string, requestedDomains []string, renewalThreshold time.Duration) (bool, string, error) {
	// Read the certificate file
	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return true, fmt.Sprintf("could not read certificate file: %v", err), err
	}

	block, _ := pem.Decode(certBytes)
	if block == nil {
		return true, "failed to decode PEM block", fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return true, fmt.Sprintf("failed to parse certificate: %v", err), err
	}

	// Check expiry
	timeLeft := time.Until(cert.NotAfter)
	expiryReason := ""
	if timeLeft <= renewalThreshold {
		expiryReason = fmt.Sprintf("certificate expires in %v (threshold is %v)",
			timeLeft.Round(time.Hour), renewalThreshold.Round(time.Hour))
		return true, expiryReason, nil
	}

	// Check for domain mismatches
	missingDomains, _ := CompareCertificateDomains(cert, requestedDomains)
	if len(missingDomains) > 0 {
		return true, fmt.Sprintf("certificate missing domains: %v", missingDomains), nil
	}

	// No renewal needed
	return false, "", nil
}

// CompareCertificateDomains compares the domains in a certificate against a list of requested domains
// Returns two slices: domains missing from the cert, and domains in cert but not requested
func CompareCertificateDomains(cert *x509.Certificate, requestedDomains []string) (missingDomains, extraDomains []string) {
	// Create maps for easier comparison
	existingDomainsMap := make(map[string]bool)
	for _, domain := range cert.DNSNames {
		existingDomainsMap[domain] = true
	}

	requestedDomainsMap := make(map[string]bool)
	for _, domain := range requestedDomains {
		requestedDomainsMap[domain] = true
	}

	// Find domains missing from certificate
	for _, domain := range requestedDomains {
		if !existingDomainsMap[domain] {
			missingDomains = append(missingDomains, domain)
		}
	}

	// Find extra domains in certificate
	for _, domain := range cert.DNSNames {
		if !requestedDomainsMap[domain] {
			extraDomains = append(extraDomains, domain)
		}
	}

	return missingDomains, extraDomains
}
