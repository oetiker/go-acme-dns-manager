package test_integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/internal/manager"
	"github.com/oetiker/go-acme-dns-manager/internal/manager/test_helpers"
	"github.com/oetiker/go-acme-dns-manager/internal/manager/test_mocks"
)

// TestEndToEndFlow tests the full certificate flow using mock servers
func TestEndToEndFlow(t *testing.T) {
	// Only run this test if explicitly requested
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to enable.")
	}

	// Start mock ACME DNS server
	mockAcmeDns := test_mocks.NewMockAcmeDnsServer()
	defer mockAcmeDns.Close()

	// Start mock ACME server
	mockAcme := test_mocks.NewMockAcmeServer()
	defer mockAcme.Close()

	// Create a temporary directory for test data
	tempDir, err := os.MkdirTemp("", "acme-dns-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test configuration
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`
email: "test@example.com"
acme_server: "` + mockAcme.GetURL() + `/directory"
key_type: "ec256"
acme_dns_server: "` + mockAcmeDns.GetURL() + `"
cert_storage_path: "` + tempDir + `"
`)
	if err := os.WriteFile(configPath, configContent, 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load the configuration
	cfg, err := manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Initialize account store
	accountsFilePath := filepath.Join(tempDir, "acme-dns-accounts.json")
	store, err := manager.NewAccountStore(accountsFilePath)
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	// Test domain
	testDomain := "test.example.com"

	// Test Steps:

	// 1. Register a new ACME DNS account
	newAccount, err := manager.RegisterNewAccount(cfg, store, testDomain)
	if err != nil {
		t.Fatalf("Failed to register ACME DNS account: %v", err)
	}
	t.Logf("Successfully registered ACME DNS account: %s -> %s", testDomain, newAccount.FullDomain)

	// 2. Verify ACME DNS account was saved
	savedAccount, exists := store.GetAccount(testDomain)
	if !exists {
		t.Fatal("Failed to save account in store")
	}
	if savedAccount.FullDomain != newAccount.FullDomain {
		t.Fatalf("Account mismatch: got %s, want %s", savedAccount.FullDomain, newAccount.FullDomain)
	}

	// 3. Create a mock DNS resolver
	mockResolver := test_mocks.NewMockDNSResolver()
	mockResolver.AddCNAMERecord("_acme-challenge."+testDomain, newAccount.FullDomain)

	// 4. Run Lego to obtain a certificate
	certName := "test-cert"
	domains := []string{testDomain}

	// Use the test_helpers.MockLegoRun function directly instead of the real RunLego
	err = test_helpers.MockLegoRun(cfg, store, "init", certName, domains, "ec256")
	if err != nil {
		t.Fatalf("Failed to run mock Lego: %v", err)
	}

	// 5. Verify certificate files exist
	certFiles := []string{
		filepath.Join(tempDir, "certificates", certName+".crt"),
		filepath.Join(tempDir, "certificates", certName+".key"),
		filepath.Join(tempDir, "certificates", certName+".json"),
	}

	for _, file := range certFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected certificate file not found: %s", file)
		}
	}

	// 6. Test certificate renewal
	// First, let's wait a moment
	time.Sleep(time.Second)

	err = test_helpers.MockLegoRun(cfg, store, "renew", certName, domains, "ec256")
	if err != nil {
		t.Fatalf("Failed to renew certificate: %v", err)
	}

	// 7. Check files were updated (timestamps should be newer)
	for _, file := range certFiles {
		info, err := os.Stat(file)
		if err != nil {
			t.Errorf("Failed to stat file after renewal: %s: %v", file, err)
			continue
		}

		// File should be recently modified
		if time.Since(info.ModTime()) > 5*time.Second {
			t.Errorf("File %s doesn't appear to have been updated during renewal", file)
		}
	}

	t.Log("End-to-end test completed successfully")
}
