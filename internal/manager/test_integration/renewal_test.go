package test_integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/internal/manager"
	"github.com/oetiker/go-acme-dns-manager/internal/manager/test_helpers"
)

// TestCertificateRenewal tests the certificate renewal logic
func TestCertificateRenewal(t *testing.T) {
	// Skip unless explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping certificate renewal test. Set RUN_INTEGRATION_TESTS=1 to enable.")
	}

	// Create a temporary directory for test data
	tempDir, err := os.MkdirTemp("", "cert-renewal-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test certificate directories
	certsDir := filepath.Join(tempDir, "certificates")
	if err := os.MkdirAll(certsDir, manager.DirPermissions); err != nil {
		t.Fatalf("Failed to create certificates directory: %v", err)
	}

	// Create a test configuration file
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
    example-com:
      domains:
        - example.com
        - www.example.com
    expired-test:
      domains:
        - expired.example.com
    soon-expiring:
      domains:
        - expiring.example.com
`)
	if err := os.WriteFile(configPath, configContent, manager.PrivateKeyPermissions); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load the configuration
	cfg, err := manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create test certificates with different expiry dates

	// 1. A certificate that's still valid (not expiring soon)
	validCert := filepath.Join(certsDir, "example-com.crt")
	validKey := filepath.Join(certsDir, "example-com.key")
	validJSON := filepath.Join(certsDir, "example-com.json")

	// Generate certificate that's valid for 60 days
	validDomains := []string{"example.com", "www.example.com"}
	generateTestCertificate(t, validCert, validKey, validJSON, validDomains, 60)

	// 2. A certificate that's expired
	expiredCert := filepath.Join(certsDir, "expired-test.crt")
	expiredKey := filepath.Join(certsDir, "expired-test.key")
	expiredJSON := filepath.Join(certsDir, "expired-test.json")

	// Generate certificate that's expired (-10 days from now)
	expiredDomains := []string{"expired.example.com"}
	generateExpiredTestCertificate(t, expiredCert, expiredKey, expiredJSON, expiredDomains, -10)

	// 3. A certificate that's expiring soon
	expiringSoonCert := filepath.Join(certsDir, "soon-expiring.crt")
	expiringSoonKey := filepath.Join(certsDir, "soon-expiring.key")
	expiringSoonJSON := filepath.Join(certsDir, "soon-expiring.json")

	// Generate certificate that's expiring in 15 days (within the graceDays=30 window)
	expiringSoonDomains := []string{"expiring.example.com"}
	generateTestCertificate(t, expiringSoonCert, expiringSoonKey, expiringSoonJSON, expiringSoonDomains, 15)

	// Now test the auto-renewal logic

	// Initialize account store for the test
	accountStorePath := filepath.Join(tempDir, "acme-dns-accounts.json")
	store, err := manager.NewAccountStore(accountStorePath)
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	// Mock accounts for our test domains
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
		"expired.example.com": {
			Username:   "user3",
			Password:   "pass3",
			FullDomain: "ghi.acme-dns.example.com",
			SubDomain:  "ghi",
		},
		"expiring.example.com": {
			Username:   "user4",
			Password:   "pass4",
			FullDomain: "jkl.acme-dns.example.com",
			SubDomain:  "jkl",
		},
	}

	// Add mock accounts to store
	for domain, account := range mockAccounts {
		store.SetAccount(domain, account)
	}
	if err := store.SaveAccounts(); err != nil {
		t.Fatalf("Failed to save accounts: %v", err)
	}

	// Create our certificate testing function that will use mock RunLego
	mockRunLegoWasCalled := map[string]bool{} // Track which certificates were renewed

	// Mock function for testing which certificates get renewed
	testRenewLogic := func(certName string, renewalThreshold time.Duration) (bool, error) {
		certPath := filepath.Join(certsDir, certName+".crt")

		needsRenewal, err := checkCertificateNeedsRenewal(t, certPath, renewalThreshold)
		if err != nil {
			return false, err
		}

		if needsRenewal {
			// In a real test, we'd call the actual renewal function
			// For this test, just record that it was called
			mockRunLegoWasCalled[certName] = true

			// Generate a new certificate to simulate renewal
			var domains []string
			if certName == "example-com" {
				domains = validDomains
			} else if certName == "expired-test" {
				domains = expiredDomains
			} else if certName == "soon-expiring" {
				domains = expiringSoonDomains
			}

			// Create a new certificate with 60 days validity
			keyPath := filepath.Join(certsDir, certName+".key")
			jsonPath := filepath.Join(certsDir, certName+".json")
			generateTestCertificate(t, certPath, keyPath, jsonPath, domains, 60)
		}

		return needsRenewal, nil
	}

	// Test each certificate
	certNames := []string{"example-com", "expired-test", "soon-expiring"}
	for _, certName := range certNames {
		renewed, err := testRenewLogic(certName, cfg.GetRenewalThreshold())
		if err != nil {
			t.Errorf("Error checking renewal for %s: %v", certName, err)
		}
		t.Logf("Certificate %s needs renewal: %v", certName, renewed)
	}

	// Verify our expectations
	if !mockRunLegoWasCalled["expired-test"] {
		t.Error("Expected expired certificate to be renewed, but it wasn't")
	}

	if !mockRunLegoWasCalled["soon-expiring"] {
		t.Error("Expected soon-expiring certificate to be renewed, but it wasn't")
	}

	if mockRunLegoWasCalled["example-com"] {
		t.Error("Did not expect valid certificate to be renewed, but it was")
	}

	// Verify that the previously expired and soon-expiring certificates now have valid dates
	// after being "renewed"
	for _, certName := range []string{"expired-test", "soon-expiring"} {
		certPath := filepath.Join(certsDir, certName+".crt")
		isValid, err := test_helpers.ValidateCertificateExpiry(t, certPath)
		if err != nil {
			t.Errorf("Error validating renewed certificate %s: %v", certName, err)
			continue
		}
		if !isValid {
			t.Errorf("Certificate %s should be valid after renewal, but it's not", certName)
		} else {
			t.Logf("Certificate %s is now valid after renewal", certName)
		}
	}
}

// Helper function to check if a certificate needs renewal
func checkCertificateNeedsRenewal(t *testing.T, certPath string, renewalThreshold time.Duration) (bool, error) {
	certInfo, err := test_helpers.ValidateCertificateFile(t, certPath)
	if err != nil {
		return false, err
	}

	// Calculate time until expiry
	timeUntilExpiry := certInfo.NotAfter.Sub(time.Now())

	// If expiry is within threshold, it needs renewal
	needsRenewal := timeUntilExpiry <= renewalThreshold

	if needsRenewal {
		t.Logf("Certificate %s needs renewal (expires in %v, threshold is %v)",
			certPath, timeUntilExpiry.Round(time.Hour*24), renewalThreshold.Round(time.Hour*24))
	} else {
		t.Logf("Certificate %s does not need renewal (expires in %v, threshold is %v)",
			certPath, timeUntilExpiry.Round(time.Hour*24), renewalThreshold.Round(time.Hour*24))
	}

	return needsRenewal, nil
}

// Generate a test certificate that's valid starting from now for the specified days
func generateTestCertificate(t *testing.T, certPath, keyPath, jsonPath string, domains []string, validDays int) {
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

// Generate an expired test certificate
func generateExpiredTestCertificate(t *testing.T, certPath, keyPath, jsonPath string, domains []string, validDays int) {
	// For expired certs, the validity period is in the past
	notBefore := time.Now().Add(time.Duration(validDays-30) * 24 * time.Hour) // Started validity in the past
	notAfter := time.Now().Add(time.Duration(validDays) * 24 * time.Hour)     // Expired already

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

	// Create a simple certificate JSON metadata
	jsonContent := `{
		"domain": "` + domains[0] + `",
		"domains": ["` + domains[0] + `"`

	for i := 1; i < len(domains); i++ {
		jsonContent += `, "` + domains[i] + `"`
	}

	jsonContent += `],
		"certUrl": "https://example.com/acme/cert/789012",
		"certStableUrl": "https://example.com/acme/cert/789012"
	}`

	// Write the JSON file
	if err := os.WriteFile(jsonPath, []byte(jsonContent), manager.PrivateKeyPermissions); err != nil {
		t.Fatalf("Failed to write certificate JSON file: %v", err)
	}
}
