//go:build testutils
// +build testutils

package test_integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// TestCertificateDomainsCheck tests that certificates are renewed when domains don't match
func TestCertificateDomainsCheck(t *testing.T) {
	// Skip unless explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping certificate domains check test. Set RUN_INTEGRATION_TESTS=1 to enable.")
	}

	// Create a temporary directory for test data
	tempDir, err := os.MkdirTemp("", "cert-domains-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temporary directory: %v", err)
		}
	}()

	// Create test certificate directories
	certsDir := filepath.Join(tempDir, "certificates")
	if err := os.MkdirAll(certsDir, manager.DirPermissions); err != nil {
		t.Fatalf("Failed to create certificates directory: %v", err)
	}

	testCases := []struct {
		name             string
		certDomains      []string
		requestedDomains []string
		expiryDays       int
		expectRenewal    bool
		renewalReason    string
	}{
		{
			name:             "Certificate with all requested domains - no renewal",
			certDomains:      []string{"example.com", "www.example.com", "api.example.com"},
			requestedDomains: []string{"example.com", "www.example.com", "api.example.com"},
			expiryDays:       90,
			expectRenewal:    false,
			renewalReason:    "",
		},
		{
			name:             "Certificate missing a requested domain - needs renewal",
			certDomains:      []string{"example.com", "www.example.com"},
			requestedDomains: []string{"example.com", "www.example.com", "api.example.com"},
			expiryDays:       90,
			expectRenewal:    true,
			renewalReason:    "certificate missing domains: [api.example.com]",
		},
		{
			name:             "Certificate with extra domains but all requested present - no renewal",
			certDomains:      []string{"example.com", "www.example.com", "api.example.com", "old.example.com"},
			requestedDomains: []string{"example.com", "www.example.com", "api.example.com"},
			expiryDays:       90,
			expectRenewal:    false,
			renewalReason:    "",
		},
		{
			name:             "Certificate missing multiple domains - needs renewal",
			certDomains:      []string{"example.com"},
			requestedDomains: []string{"example.com", "www.example.com", "api.example.com", "shop.example.com"},
			expiryDays:       90,
			expectRenewal:    true,
			renewalReason:    "certificate missing domains: [www.example.com api.example.com shop.example.com]",
		},
		{
			name:             "Valid certificate but near expiry - needs renewal",
			certDomains:      []string{"example.com", "www.example.com"},
			requestedDomains: []string{"example.com", "www.example.com"},
			expiryDays:       20, // Less than typical 30-day threshold
			expectRenewal:    true,
			renewalReason:    "certificate expires in",
		},
		{
			name:             "Certificate with wildcard matching base domain",
			certDomains:      []string{"example.com", "*.example.com"},
			requestedDomains: []string{"example.com", "*.example.com"},
			expiryDays:       90,
			expectRenewal:    false,
			renewalReason:    "",
		},
		{
			name:             "Certificate missing wildcard domain",
			certDomains:      []string{"example.com"},
			requestedDomains: []string{"example.com", "*.example.com"},
			expiryDays:       90,
			expectRenewal:    true,
			renewalReason:    "certificate missing domains: [*.example.com]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test certificate with specific domains
			certPath := filepath.Join(certsDir, "test.example.com.crt")
			err := createTestCertificate(certPath, tc.certDomains, tc.expiryDays)
			if err != nil {
				t.Fatalf("Failed to create test certificate: %v", err)
			}

			// Check if certificate needs renewal
			renewalThreshold := 30 * 24 * time.Hour
			needsRenewal, reason, err := manager.CertificateNeedsRenewal(certPath, tc.requestedDomains, renewalThreshold)

			// Check results
			if needsRenewal != tc.expectRenewal {
				t.Errorf("Expected needsRenewal=%v but got %v", tc.expectRenewal, needsRenewal)
			}

			if tc.renewalReason != "" && reason == "" {
				t.Errorf("Expected renewal reason containing %q but got empty reason", tc.renewalReason)
			}

			if tc.renewalReason != "" && reason != "" {
				// For expiry reason, just check if it contains the expected prefix
				if tc.renewalReason == "certificate expires in" {
					if len(reason) < len(tc.renewalReason) || reason[:len(tc.renewalReason)] != tc.renewalReason {
						t.Errorf("Expected renewal reason starting with %q but got %q", tc.renewalReason, reason)
					}
				} else if reason != tc.renewalReason {
					t.Errorf("Expected renewal reason %q but got %q", tc.renewalReason, reason)
				}
			}

			if !tc.expectRenewal && reason != "" {
				t.Errorf("Expected no renewal reason but got %q", reason)
			}
		})
	}
}

// createTestCertificate creates a test certificate with specified domains and expiry
func createTestCertificate(certPath string, domains []string, daysUntilExpiry int) error {
	// Generate a private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Organization"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Duration(daysUntilExpiry) * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Set the DNS names
	if len(domains) > 0 {
		template.DNSNames = domains
		// Use first domain as CN
		template.Subject.CommonName = domains[0]
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// Write certificate to file
	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()

	// Write the certificate in PEM format
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err != nil {
		return err
	}

	return nil
}

// TestCompareCertificateDomainsIntegration tests the domain comparison with real certificates
func TestCompareCertificateDomainsIntegration(t *testing.T) {
	// Skip unless explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping domain comparison integration test. Set RUN_INTEGRATION_TESTS=1 to enable.")
	}

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "domain-compare-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test certificate with specific domains
	certPath := filepath.Join(tempDir, "test.crt")
	testDomains := []string{"example.com", "www.example.com", "api.example.com"}
	err = createTestCertificate(certPath, testDomains, 90)
	if err != nil {
		t.Fatalf("Failed to create test certificate: %v", err)
	}

	// Read and parse the certificate
	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("Failed to read certificate: %v", err)
	}

	block, _ := pem.Decode(certBytes)
	if block == nil {
		t.Fatalf("Failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	// Test various domain comparison scenarios
	testCases := []struct {
		name            string
		requestedDomains []string
		expectedMissing  []string
		expectedExtra    []string
	}{
		{
			name:            "Exact match",
			requestedDomains: []string{"example.com", "www.example.com", "api.example.com"},
			expectedMissing:  []string{},
			expectedExtra:    []string{},
		},
		{
			name:            "Missing domain",
			requestedDomains: []string{"example.com", "www.example.com", "api.example.com", "new.example.com"},
			expectedMissing:  []string{"new.example.com"},
			expectedExtra:    []string{},
		},
		{
			name:            "Extra domain in cert",
			requestedDomains: []string{"example.com", "www.example.com"},
			expectedMissing:  []string{},
			expectedExtra:    []string{"api.example.com"},
		},
		{
			name:            "Both missing and extra",
			requestedDomains: []string{"example.com", "new.example.com", "another.example.com"},
			expectedMissing:  []string{"new.example.com", "another.example.com"},
			expectedExtra:    []string{"www.example.com", "api.example.com"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			missing, extra := manager.CompareCertificateDomains(cert, tc.requestedDomains)

			// Check missing domains
			if len(missing) != len(tc.expectedMissing) {
				t.Errorf("Expected %d missing domains, got %d: %v", len(tc.expectedMissing), len(missing), missing)
			}
			for _, expectedDomain := range tc.expectedMissing {
				found := false
				for _, missingDomain := range missing {
					if missingDomain == expectedDomain {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected missing domain %q not found in result", expectedDomain)
				}
			}

			// Check extra domains
			if len(extra) != len(tc.expectedExtra) {
				t.Errorf("Expected %d extra domains, got %d: %v", len(tc.expectedExtra), len(extra), extra)
			}
			for _, expectedDomain := range tc.expectedExtra {
				found := false
				for _, extraDomain := range extra {
					if extraDomain == expectedDomain {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected extra domain %q not found in result", expectedDomain)
				}
			}
		})
	}
}
