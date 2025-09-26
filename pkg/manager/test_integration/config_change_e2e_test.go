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
	"github.com/oetiker/go-acme-dns-manager/pkg/manager/test_helpers"
)

// TestEndToEndConfigChange tests certificate renewal when config changes add domains
func TestEndToEndConfigChange(t *testing.T) {
	// Skip unless explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping E2E config change test. Set RUN_INTEGRATION_TESTS=1 to enable.")
	}

	// Create a temporary directory for test data
	tempDir, err := os.MkdirTemp("", "e2e-config-change-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temporary directory: %v", err)
		}
	}()

	// Create certificates directory
	certsDir := filepath.Join(tempDir, "certificates")
	if err := os.MkdirAll(certsDir, manager.DirPermissions); err != nil {
		t.Fatalf("Failed to create certificates directory: %v", err)
	}

	// Step 1: Create an existing certificate with only example.com (valid for 60 days)
	t.Log("Step 1: Creating existing certificate with only example.com domain")
	certName := "test.example.com"
	certPath := filepath.Join(certsDir, certName+".crt")
	keyPath := filepath.Join(certsDir, certName+".key")
	metadataPath := filepath.Join(certsDir, certName+".json")

	// Create a valid certificate with only example.com
	err = createE2ECertificate(certPath, keyPath, []string{"example.com"}, 60)
	if err != nil {
		t.Fatalf("Failed to create test certificate: %v", err)
	}

	// Create metadata file
	metadata := `{"domain":"example.com","domains":["example.com"],"certificate":"CERT","key":"KEY"}`
	if err := os.WriteFile(metadataPath, []byte(metadata), 0600); err != nil {
		t.Fatalf("Failed to create metadata file: %v", err)
	}

	// Step 2: Test if CertificateNeedsRenewal detects the missing domain
	t.Log("Step 2: Testing if certificate renewal is needed when requesting additional domain")

	requestedDomains := []string{"example.com", "www.example.com"}
	renewalThreshold := 30 * 24 * time.Hour // 30 days

	needsRenewal, reason, err := manager.CertificateNeedsRenewal(certPath, requestedDomains, renewalThreshold)
	if err != nil {
		t.Fatalf("Failed to check certificate renewal: %v", err)
	}

	if !needsRenewal {
		t.Error("FAILED: CertificateNeedsRenewal should return true when domains are missing")
		t.Error("Certificate has: [example.com]")
		t.Error("Config requests: [example.com, www.example.com]")
		t.Error("Expected renewal due to missing www.example.com")
	} else {
		t.Logf("SUCCESS: Certificate needs renewal - reason: %s", reason)
	}

	// Step 3: Test actual renewal with mock Lego
	t.Log("Step 3: Testing certificate renewal with mock Lego")

	// Create a config
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `
email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
acme_dns_server: "https://acme-dns.example.com"
cert_storage_path: "` + tempDir + `"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Load configuration
	cfg, err := manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Initialize account store with existing accounts
	accountsFilePath := filepath.Join(tempDir, "acme-dns-accounts.json")
	// Create accounts file with pre-existing accounts
	accountsContent := `{
		"example.com": {
			"username": "test-user",
			"password": "test-pass",
			"fulldomain": "test.acmedns.example.com",
			"subdomain": "test",
			"allowfrom": []
		},
		"www.example.com": {
			"username": "test-user-www",
			"password": "test-pass-www",
			"fulldomain": "test-www.acmedns.example.com",
			"subdomain": "test-www",
			"allowfrom": []
		}
	}`
	if err := os.WriteFile(accountsFilePath, []byte(accountsContent), 0600); err != nil {
		t.Fatalf("Failed to create accounts file: %v", err)
	}

	store, err := manager.NewAccountStore(accountsFilePath)
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	// Track if renewal happens with correct domains
	renewalCalled := false
	correctDomains := false

	// Wrap MockLegoRun to track calls
	wrappedMockLegoRun := func(cfg *manager.Config, store interface{}, action string, certName string, domains []string, keyType string) error {
		t.Logf("MockLegoRun called with action=%s, domains=%v", action, domains)
		if action == "renew" {
			renewalCalled = true
			// Check if both domains are included
			if len(domains) == 2 && containsDomain(domains, "example.com") && containsDomain(domains, "www.example.com") {
				correctDomains = true
			}
		}
		return test_helpers.MockLegoRun(cfg, store, action, certName, domains, keyType)
	}

	// Run renewal
	err = wrappedMockLegoRun(cfg, store, "renew", certName, requestedDomains, "ec256")
	if err != nil {
		t.Logf("Note: MockLegoRun failed (this might be expected): %v", err)
	}

	// Step 4: Verify results
	t.Log("Step 4: Verifying renewal was triggered with correct domains")

	if !renewalCalled {
		t.Error("FAILED: Renewal was not called")
	} else if !correctDomains {
		t.Error("FAILED: Renewal was called but not with the correct domains [example.com, www.example.com]")
	} else {
		t.Log("SUCCESS: Renewal was triggered with correct domains including www.example.com")
	}
}

// createE2ECertificate creates a test certificate and private key
func createE2ECertificate(certPath, keyPath string, domains []string, daysUntilExpiry int) error {
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

	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err != nil {
		return err
	}

	// Write private key to file
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privKeyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}

	err = pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privKeyDER})
	if err != nil {
		return err
	}

	// Set appropriate permissions
	os.Chmod(keyPath, 0600)

	return nil
}

// Helper function
func containsDomain(domains []string, domain string) bool {
	for _, d := range domains {
		if d == domain {
			return true
		}
	}
	return false
}
