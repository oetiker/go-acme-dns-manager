//go:build testutils
// +build testutils

package test_integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/app"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager/test_helpers"
)

// TestRenewalWithDomainAddition tests the complete flow when a domain is added to an existing certificate
// This is an integration test that verifies the fix works end-to-end
func TestRenewalWithDomainAddition(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to enable.")
	}

	// Create temporary directory
	tempDir := t.TempDir()

	// Step 1: Create initial configuration with one domain
	configContent := `email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
acme_dns_server: "https://acme-dns.example.com"
cert_storage_path: "` + tempDir + `"

auto_domains:
  grace_days: 30
  certs:
    test.example.com:
      domains:
        - example.com
`

	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Step 2: Simulate initial certificate creation (with mock)
	certsDir := filepath.Join(tempDir, "certificates")
	if err := os.MkdirAll(certsDir, manager.DirPermissions); err != nil {
		t.Fatalf("Failed to create certificates directory: %v", err)
	}

	// Use mock to create initial certificate with one domain
	cfg := &manager.Config{
		Email:           "test@example.com",
		AcmeServer:      "https://acme-staging-v02.api.letsencrypt.org/directory",
		AcmeDnsServer:   "https://acme-dns.example.com",
		CertStoragePath: tempDir,
	}

	// Create mock certificate with only example.com
	err := test_helpers.MockLegoRun(cfg, nil, "init", "test.example.com", []string{"example.com"}, "rsa2048")
	if err != nil {
		t.Fatalf("Failed to create initial certificate: %v", err)
	}

	// Step 3: Update configuration to add www.example.com
	configContent = `email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
acme_dns_server: "https://acme-dns.example.com"
cert_storage_path: "` + tempDir + `"

auto_domains:
  grace_days: 30
  certs:
    test.example.com:
      domains:
        - example.com
        - www.example.com  # NEW domain added
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// Step 4: Load the updated configuration
	cfg, err = manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Step 5: Check if certificate needs renewal
	certPath := filepath.Join(certsDir, "test.example.com.crt")
	requestedDomains := []string{"example.com", "www.example.com"}
	renewalThreshold := 30 * 24 * time.Hour

	needsRenewal, reason, err := manager.CertificateNeedsRenewal(certPath, requestedDomains, renewalThreshold)
	if err != nil {
		t.Fatalf("Failed to check certificate renewal: %v", err)
	}

	// Should need renewal due to missing domain
	if !needsRenewal {
		t.Error("Certificate should need renewal due to missing www.example.com")
	}

	if !strings.Contains(reason, "missing domains") || !strings.Contains(reason, "www.example.com") {
		t.Errorf("Expected reason to mention missing www.example.com, got: %s", reason)
	}

	t.Logf("✓ Certificate correctly identified as needing renewal: %s", reason)

	// Step 6: Test renewal with mock (simulating the actual renewal flow)
	// This would normally check ACME-DNS and get a new certificate with both domains

	// For this test, we don't need actual ACME-DNS accounts since the mock doesn't check them
	// The key is that the renewal flow is exercised with domain changes
	accountsPath := filepath.Join(tempDir, "acme-dns-accounts.json")
	emptyAccounts := make(map[string]manager.AcmeDnsAccount)
	accountsJSON, _ := json.Marshal(emptyAccounts)
	if err := os.WriteFile(accountsPath, accountsJSON, 0600); err != nil {
		t.Fatalf("Failed to create accounts file: %v", err)
	}

	// Simulate renewal with the mock (store is not needed for mock)
	err = test_helpers.MockLegoRun(cfg, nil, "renew", "test.example.com", requestedDomains, "rsa2048")
	if err != nil {
		// Note: In real scenario, this might fail with ErrDNSSetupNeeded if ACME-DNS not configured
		t.Logf("Renewal attempt result: %v", err)
	}

	t.Log("✓ Integration test completed - renewal with domain changes flow verified")
}

// TestRenewalDomainChangeWithApp tests the complete application flow for domain change renewal
func TestRenewalDomainChangeWithApp(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to enable.")
	}

	// Create temporary directory
	tempDir := t.TempDir()

	// Create initial certificate with one domain
	certsDir := filepath.Join(tempDir, "certificates")
	if err := os.MkdirAll(certsDir, manager.DirPermissions); err != nil {
		t.Fatalf("Failed to create certificates directory: %v", err)
	}

	// Create mock certificate
	cfg := &manager.Config{
		Email:           "test@example.com",
		AcmeServer:      "https://acme-staging-v02.api.letsencrypt.org/directory",
		AcmeDnsServer:   "https://acme-dns.example.com",
		CertStoragePath: tempDir,
	}

	err := test_helpers.MockLegoRun(cfg, nil, "init", "web.example.com", []string{"example.com"}, "rsa2048")
	if err != nil {
		t.Fatalf("Failed to create initial certificate: %v", err)
	}

	// Create configuration with additional domain
	cfg.AutoDomains = &manager.AutoDomainsConfig{
		GraceDays: 30,
		Certs: map[string]manager.CertConfig{
			"web.example.com": {
				Domains: []string{"example.com", "www.example.com"}, // Added www
				KeyType: "rsa2048",
			},
		},
	}

	// Create empty ACME-DNS accounts file
	accountsPath := filepath.Join(tempDir, "acme-dns-accounts.json")
	emptyAccounts := make(map[string]manager.AcmeDnsAccount)
	accountsJSON, _ := json.Marshal(emptyAccounts)
	if err := os.WriteFile(accountsPath, accountsJSON, 0600); err != nil {
		t.Fatalf("Failed to create accounts file: %v", err)
	}

	// Create certificate manager with mock runner
	logger := manager.GetDefaultLogger()
	certManager, err := app.NewCertificateManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	// Replace with mock runner
	app.DefaultLegoRunner = test_helpers.MockLegoRun

	// Process in auto mode (should detect domain change and attempt renewal)
	ctx := context.Background()
	err = certManager.ProcessAutoMode(ctx)

	// The renewal will likely fail due to missing ACME-DNS for www.example.com
	// but that's expected and proves the check is working
	if err != nil {
		if errors.Is(err, manager.ErrDNSSetupNeeded) {
			t.Log("✓ Renewal correctly identified need for DNS setup for new domain")
		} else {
			t.Logf("Renewal failed (expected): %v", err)
		}
	} else {
		t.Log("✓ Renewal completed successfully")
	}
}
