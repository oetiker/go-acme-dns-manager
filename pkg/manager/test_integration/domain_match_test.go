package test_integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager/test_helpers"
)

// TestCertificateDomainMatch specifically tests the logic for determining
// if a certificate needs renewal based on whether all required domains are included
// in the certificate.
func TestCertificateDomainMatch(t *testing.T) {
	// Skip unless explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping certificate domain match test. Set RUN_INTEGRATION_TESTS=1 to enable.")
	}

	// Create a temporary directory for test data
	tempDir, err := os.MkdirTemp("", "cert-domain-match-test")
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

	// Create test configuration with 30 days grace period
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`
email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
key_type: "ec256"
acme_dns_server: "https://acme-dns.example.com"
cert_storage_path: "` + tempDir + `"

# Auto domains configuration for testing
autoDomains:
  graceDays: 30
  certs:
    match-test:
      domains:
        - example.com
        - www.example.com
    mismatch-test:
      domains:
        - example.org
        - www.example.org
        - api.example.org
    partial-test:
      domains:
        - example.net
        - www.example.net
        - api.example.net
`)
	if err := os.WriteFile(configPath, configContent, manager.PrivateKeyPermissions); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load the configuration - we don't need it for this test but keeping for future use
	_, err = manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create three different test certificates:
	// 1. A certificate with domains that exactly match the config
	matchCert := filepath.Join(certsDir, "match-test.crt")
	matchKey := filepath.Join(certsDir, "match-test.key")
	matchJSON := filepath.Join(certsDir, "match-test.json")
	matchDomains := []string{"example.com", "www.example.com"}
	generateDomainTestCertificate(t, matchCert, matchKey, matchJSON, matchDomains, 60) // Valid for 60 days

	// 2. A certificate with domains that don't match the config (completely different domains)
	mismatchCert := filepath.Join(certsDir, "mismatch-test.crt")
	mismatchKey := filepath.Join(certsDir, "mismatch-test.key")
	mismatchJSON := filepath.Join(certsDir, "mismatch-test.json")
	mismatchDomains := []string{"different.org", "sub.different.org"} // Different domains than in config
	generateDomainTestCertificate(t, mismatchCert, mismatchKey, mismatchJSON, mismatchDomains, 60)

	// 3. A certificate with some domains matching config but some missing
	partialCert := filepath.Join(certsDir, "partial-test.crt")
	partialKey := filepath.Join(certsDir, "partial-test.key")
	partialJSON := filepath.Join(certsDir, "partial-test.json")
	partialDomains := []string{"example.net", "www.example.net"} // Missing api.example.net
	generateDomainTestCertificate(t, partialCert, partialKey, partialJSON, partialDomains, 60)

	// Initialize account store for the test
	accountStorePath := filepath.Join(tempDir, "acme-dns-accounts.json")
	store, err := manager.NewAccountStore(accountStorePath)
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	// Add mock accounts for our test domains
	mockAccounts := map[string]manager.AcmeDnsAccount{
		"example.com": {
			Username:   "user1",
			Password:   "pass1",
			FullDomain: "abc.acme-dns.example.com",
			SubDomain:  "abc",
		},
		"www.example.com": {
			Username:   "user2",
			Password:   "pass2",
			FullDomain: "def.acme-dns.example.com",
			SubDomain:  "def",
		},
		"example.org": {
			Username:   "user3",
			Password:   "pass3",
			FullDomain: "ghi.acme-dns.example.com",
			SubDomain:  "ghi",
		},
		"www.example.org": {
			Username:   "user4",
			Password:   "pass4",
			FullDomain: "jkl.acme-dns.example.com",
			SubDomain:  "jkl",
		},
		"api.example.org": {
			Username:   "user5",
			Password:   "pass5",
			FullDomain: "mno.acme-dns.example.com",
			SubDomain:  "mno",
		},
		"example.net": {
			Username:   "user6",
			Password:   "pass6",
			FullDomain: "pqr.acme-dns.example.com",
			SubDomain:  "pqr",
		},
		"www.example.net": {
			Username:   "user7",
			Password:   "pass7",
			FullDomain: "stu.acme-dns.example.com",
			SubDomain:  "stu",
		},
		"api.example.net": {
			Username:   "user8",
			Password:   "pass8",
			FullDomain: "vwx.acme-dns.example.com",
			SubDomain:  "vwx",
		},
	}

	// Add mock accounts to store
	for domain, account := range mockAccounts {
		store.SetAccount(domain, account)
	}
	if err := store.SaveAccounts(); err != nil {
		t.Fatalf("Failed to save accounts: %v", err)
	}

	// Test the domain match checking logic
	type testCase struct {
		name            string
		certPath        string
		configDomains   []string
		shouldNeedRenew bool
	}

	testCases := []testCase{
		{
			name:            "Matching domains",
			certPath:        matchCert,
			configDomains:   []string{"example.com", "www.example.com"},
			shouldNeedRenew: false, // Should not need renewal as all domains are present
		},
		{
			name:            "Completely different domains",
			certPath:        mismatchCert,
			configDomains:   []string{"example.org", "www.example.org", "api.example.org"},
			shouldNeedRenew: true, // Should need renewal as domains are completely different
		},
		{
			name:            "Missing one domain from config",
			certPath:        partialCert,
			configDomains:   []string{"example.net", "www.example.net", "api.example.net"},
			shouldNeedRenew: true, // Should need renewal as one domain is missing
		},
	}

	// Mock function to check if certificate needs renewal based on domain match
	checkDomainMatch := func(certPath string, configDomains []string) (bool, error) {
		// Load certificate file
		certInfo, err := test_helpers.ValidateCertificateFile(t, certPath)
		if err != nil {
			return false, err
		}

		// Create map of certificate domains
		certDomains := make(map[string]bool)
		for _, domain := range certInfo.DNSNames {
			certDomains[domain] = true
		}

		// Also add CommonName to the domain map if it's not in DNSNames
		if certInfo.CommonName != "" && !certDomains[certInfo.CommonName] {
			certDomains[certInfo.CommonName] = true
		}

		// Check if all requested domains are in the certificate
		var missingDomains []string
		for _, domain := range configDomains {
			if !certDomains[domain] {
				missingDomains = append(missingDomains, domain)
			}
		}

		// If any domains are missing from the certificate, renewal is needed
		needsRenewal := len(missingDomains) > 0

		if needsRenewal {
			t.Logf("Certificate %s needs renewal due to missing domains: %v", certPath, missingDomains)
			t.Logf("   Certificate has domains: %v", certInfo.DNSNames)
			t.Logf("   Config requests domains: %v", configDomains)
		} else {
			t.Logf("Certificate %s has all required domains", certPath)
			t.Logf("   Certificate has domains: %v", certInfo.DNSNames)
			t.Logf("   Config requests domains: %v", configDomains)
		}

		return needsRenewal, nil
	}

	// Run the tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			needsRenewal, err := checkDomainMatch(tc.certPath, tc.configDomains)
			if err != nil {
				t.Fatalf("Error checking domain match: %v", err)
			}

			if needsRenewal != tc.shouldNeedRenew {
				if tc.shouldNeedRenew {
					t.Errorf("Certificate should need renewal due to domain mismatch but doesn't")
				} else {
					t.Errorf("Certificate should not need renewal but is marked as needing renewal")
				}
			}
		})
	}

	// Bonus: Test with ValidateCertificateDomains from test_helpers package
	t.Run("Using ValidateCertificateDomains", func(t *testing.T) {
		// Test for the partial match case which should clearly show missing domains
		isValid, err := test_helpers.ValidateCertificateDomains(t, partialCert,
			[]string{"example.net", "www.example.net", "api.example.net"})

		if err != nil {
			t.Fatalf("Error validating domains: %v", err)
		}

		if isValid {
			t.Errorf("ValidateCertificateDomains should have returned false for missing domains")
		}
	})
}

// Generate a test certificate for domain matching tests
func generateDomainTestCertificate(t *testing.T, certPath, keyPath, jsonPath string, domains []string, validDays int) {
	// Create a simple certificate JSON metadata
	jsonContent := `{
		"domain": "` + domains[0] + `",
		"domains": ["` + domains[0] + `"`

	for i := 1; i < len(domains); i++ {
		jsonContent += `, "` + domains[i] + `"`
	}

	jsonContent += `],
		"certUrl": "https://example.com/acme/cert/123456",
		"certStableUrl": "https://example.com/acme/cert/123456"
	}`

	// Write the JSON file
	if err := os.WriteFile(jsonPath, []byte(jsonContent), manager.PrivateKeyPermissions); err != nil {
		t.Fatalf("Failed to write certificate JSON file: %v", err)
	}

	// Generate a simple self-signed certificate
	notBefore := time.Now().Add(-24 * time.Hour)                          // Valid from yesterday
	notAfter := time.Now().Add(time.Duration(validDays) * 24 * time.Hour) // Valid for specified days

	// Create test certificates
	certPEM, keyPEM, err := test_helpers.GenerateSelfSignedCert(domains[0], domains, notBefore, notAfter)
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	// Write the certificate and key files
	if err := os.WriteFile(certPath, certPEM, manager.CertificatePermissions); err != nil {
		t.Fatalf("Failed to write certificate file: %v", err)
	}

	if err := os.WriteFile(keyPath, keyPEM, manager.PrivateKeyPermissions); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}
}
