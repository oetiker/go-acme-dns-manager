package manager

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "acmedns-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid config file
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`
email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
key_type: "ec256"
acme_dns_server: "https://acme-dns.example.com"
cert_storage_path: ".lego"
autoDomains:
  graceDays: 30
  certs:
    test-cert:
      domains:
        - example.com
        - www.example.com
`)
	err = os.WriteFile(configPath, configContent, PrivateKeyPermissions)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Check values
	if cfg.Email != "test@example.com" {
		t.Errorf("Expected Email to be 'test@example.com', got '%s'", cfg.Email)
	}
	if cfg.AcmeServer != "https://acme-staging-v02.api.letsencrypt.org/directory" {
		t.Errorf("Expected AcmeServer to be 'https://acme-staging-v02.api.letsencrypt.org/directory', got '%s'", cfg.AcmeServer)
	}
	if cfg.KeyType != "ec256" {
		t.Errorf("Expected KeyType to be 'ec256', got '%s'", cfg.KeyType)
	}

	// Test autoDomains parsing
	if cfg.AutoDomains == nil {
		t.Fatal("Expected AutoDomains to be non-nil")
	}
	if cfg.AutoDomains.GraceDays != 30 {
		t.Errorf("Expected AutoDomains.GraceDays to be 30, got %d", cfg.AutoDomains.GraceDays)
	}
	if len(cfg.AutoDomains.Certs) != 1 {
		t.Fatalf("Expected 1 cert in AutoDomains.Certs, got %d", len(cfg.AutoDomains.Certs))
	}

	// Test cert domains
	cert, ok := cfg.AutoDomains.Certs["test-cert"]
	if !ok {
		t.Fatal("Expected to find 'test-cert' in AutoDomains.Certs")
	}
	if len(cert.Domains) != 2 {
		t.Fatalf("Expected 2 domains in test-cert, got %d", len(cert.Domains))
	}
	if cert.Domains[0] != "example.com" || cert.Domains[1] != "www.example.com" {
		t.Errorf("Unexpected domains in test-cert: %v", cert.Domains)
	}
}

func TestGenerateDefaultConfig(t *testing.T) {
	var buf bytes.Buffer
	err := GenerateDefaultConfig(&buf)
	if err != nil {
		t.Fatalf("GenerateDefaultConfig failed: %v", err)
	}

	content := buf.String()
	if len(content) == 0 {
		t.Fatal("Expected non-empty default config")
	}

	// Check for expected strings in the config
	expectedStrings := []string{
		"email:", "acme_server:", "key_type:", "acme_dns_server:",
		"cert_storage_path:", "autoDomains:", "graceDays:",
	}

	for _, s := range expectedStrings {
		if !bytes.Contains(buf.Bytes(), []byte(s)) {
			t.Errorf("Expected default config to contain '%s'", s)
		}
	}
}

func TestAccountStoreOperations(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "acmedns-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a store with a temporary file path
	storePath := filepath.Join(tempDir, "accounts.json")
	store, err := NewAccountStore(storePath)
	if err != nil {
		t.Fatalf("NewAccountStore failed: %v", err)
	}

	// Test setting and getting an account
	testAccount := AcmeDnsAccount{
		Username:   "testuser",
		Password:   "testpass",
		FullDomain: "test.acme-dns.example.com",
		SubDomain:  "test",
	}

	store.SetAccount("example.com", testAccount)

	// Test retrieval
	retrieved, exists := store.GetAccount("example.com")
	if !exists {
		t.Fatal("Expected account to exist")
	}

	if retrieved.Username != testAccount.Username ||
		retrieved.Password != testAccount.Password ||
		retrieved.FullDomain != testAccount.FullDomain ||
		retrieved.SubDomain != testAccount.SubDomain {
		t.Errorf("Retrieved account doesn't match what was set")
	}

	// Test saving
	err = store.SaveAccounts()
	if err != nil {
		t.Fatalf("SaveAccounts failed: %v", err)
	}

	// Test loading from file
	newStore, err := NewAccountStore(storePath)
	if err != nil {
		t.Fatalf("NewAccountStore (reload) failed: %v", err)
	}

	// Check if account was loaded correctly
	reloaded, exists := newStore.GetAccount("example.com")
	if !exists {
		t.Fatal("Expected account to exist after reload")
	}

	if reloaded.Username != testAccount.Username {
		t.Errorf("Reloaded account username doesn't match: got %s, want %s",
			reloaded.Username, testAccount.Username)
	}
}
