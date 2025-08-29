//go:build testutils

package test_integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oetiker/go-acme-dns-manager/pkg/app"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager/test_helpers"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager/test_mocks"
)

// TestUserWorkflow_BasicCommandLine validates the basic command-line workflow
// This is a simplified version focusing on the core workflow validation
func TestUserWorkflow_BasicCommandLine(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test (set RUN_INTEGRATION_TESTS=1 to run)")
	}

	// Setup test environment
	tmpDir := t.TempDir()
	certStoragePath := filepath.Join(tmpDir, "certs")

	// Create minimal config
	config := &manager.Config{
		Email:           "test@example.com",
		AcmeServer:      "https://acme-staging-v02.api.letsencrypt.org/directory",
		AcmeDnsServer:   "https://acme-dns.example.com",
		CertStoragePath: certStoragePath,
	}

	// Set up some accounts in the store BEFORE creating certificate manager
	// to avoid registration calls, but make DNS verification fail initially
	accountsFilePath := filepath.Join(certStoragePath, "acme-dns-accounts.json")
	store, err := manager.NewAccountStore(accountsFilePath)
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	// Add a dummy account for example.com that will fail DNS verification
	store.SetAccount("example.com", manager.AcmeDnsAccount{
		Username:   "test-user",
		Password:   "test-pass",
		FullDomain: "test-uuid.acme-dns.example.com",
		SubDomain: "test-uuid",
		AllowFrom:  []string{"0.0.0.0/0"},
	})
	store.SaveAccounts()

	// Create logger
	logger := manager.NewColorfulLogger(os.Stdout, manager.LogLevelDebug, false, false)

	// Create certificate manager
	certManager, err := app.NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	// Set up mock DNS resolver that will fail verification for first run
	mockResolver := test_mocks.NewMockDNSResolver()
	// Don't add any CNAME records, so verification will fail
	certManager.SetDNSResolver(mockResolver)

	testRuns := 0
	certManager.SetLegoRunner(func(cfg *manager.Config, store interface{}, action string, certName string, domains []string, keyType string) error {
		testRuns++
		t.Logf("Mock Lego called: run=%d, action=%s, cert=%s, domains=%v", testRuns, action, certName, domains)

		if testRuns == 1 {
			// First run - DNS not configured
			if action != "init" {
				t.Errorf("Expected action 'init' on first run, got: %s", action)
			}
			return manager.ErrDNSSetupNeeded
		}

		// Second run - DNS configured
		if action != "init" {
			t.Errorf("Expected action 'init' on second run, got: %s", action)
		}

		// Simulate successful certificate creation
		certPath := filepath.Join(cfg.CertStoragePath, "certificates", certName+".crt")
		os.MkdirAll(filepath.Dir(certPath), 0755)
		test_helpers.CreateTestCertificate(t, certPath, domains, 90)

		return nil
	})

	// WORKFLOW STEP 1: First run - should get DNS setup needed
	t.Log("WORKFLOW: Step 1 - Initial run without DNS configured")
	args := []string{"test-cert@example.com"}
	ctx := context.Background()
	err = certManager.ProcessManualMode(ctx, args)

	// The error may be wrapped, so use errors.Is
	if !errors.Is(err, manager.ErrDNSSetupNeeded) {
		t.Errorf("WORKFLOW VIOLATION: Expected ErrDNSSetupNeeded on first run, got: %v", err)
	}

	// WORKFLOW STEP 2: Second run - should succeed
	t.Log("WORKFLOW: Step 2 - Run after DNS configuration")

	// Create new certificate manager for second run (simulating new invocation)
	certManager2, err := app.NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager for second run: %v", err)
	}

	// Set up mock DNS resolver with proper CNAME records (simulating user configured DNS)
	mockResolver2 := test_mocks.NewMockDNSResolver()
	mockResolver2.AddCNAMERecord("_acme-challenge.example.com", "test-uuid.acme-dns.example.com")
	certManager2.SetDNSResolver(mockResolver2)

	certManager2.SetLegoRunner(func(cfg *manager.Config, store interface{}, action string, certName string, domains []string, keyType string) error {
		// Second run - DNS is configured, should proceed normally
		if action != "init" {
			t.Errorf("Expected action 'init', got: %s", action)
		}

		// Simulate successful certificate creation
		certPath := filepath.Join(cfg.CertStoragePath, "certificates", certName+".crt")
		os.MkdirAll(filepath.Dir(certPath), 0755)
		test_helpers.CreateTestCertificate(t, certPath, domains, 90)

		return nil
	})

	err = certManager2.ProcessManualMode(ctx, args)
	if err != nil {
		t.Errorf("WORKFLOW VIOLATION: Second run should succeed, got error: %v", err)
	}

	// Verify certificate was created
	certPath := filepath.Join(certStoragePath, "certificates", "test-cert.crt")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Error("WORKFLOW VIOLATION: Certificate should exist after successful second run")
	}

	t.Log("✓ Basic command-line workflow validated successfully")
}

// TestUserWorkflow_BasicAutoMode validates the basic auto mode workflow
func TestUserWorkflow_BasicAutoMode(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test (set RUN_INTEGRATION_TESTS=1 to run)")
	}

	// Setup test environment
	tmpDir := t.TempDir()
	certStoragePath := filepath.Join(tmpDir, "certs")

	// Create config with auto_domains
	config := &manager.Config{
		Email:           "test@example.com",
		AcmeServer:      "https://acme-staging-v02.api.letsencrypt.org/directory",
		AcmeDnsServer:   "https://acme-dns.example.com",
		CertStoragePath: certStoragePath,
		AutoDomains: &manager.AutoDomainsConfig{
			GraceDays: 30,
			Certs: map[string]manager.CertConfig{
				"web-cert": {
					Domains: []string{"example.com", "www.example.com"},
				},
			},
		},
	}

	// Create logger
	logger := manager.NewColorfulLogger(os.Stdout, manager.LogLevelDebug, false, false)

	// Set up some accounts in the store to avoid registration calls, but make DNS verification fail
	accountsFilePath := filepath.Join(certStoragePath, "acme-dns-accounts.json")
	store, err := manager.NewAccountStore(accountsFilePath)
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	// Add dummy accounts for ALL domains that will fail DNS verification
	// Need to add accounts for all domains to prevent registration attempts
	store.SetAccount("example.com", manager.AcmeDnsAccount{
		Username:   "test-user",
		Password:   "test-pass",
		FullDomain: "test-uuid.acme-dns.example.com",
		SubDomain: "test-uuid",
		AllowFrom:  []string{"0.0.0.0/0"},
	})
	store.SetAccount("www.example.com", manager.AcmeDnsAccount{
		Username:   "test-user-www",
		Password:   "test-pass-www",
		FullDomain: "test-uuid-www.acme-dns.example.com",
		SubDomain: "test-uuid-www",
		AllowFrom:  []string{"0.0.0.0/0"},
	})
	store.SaveAccounts()

	// WORKFLOW STEP 1: First run - should get DNS setup needed
	t.Log("WORKFLOW: Step 1 - Initial auto mode run")

	certManager, err := app.NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	// Set up mock DNS resolver that will fail verification for first run
	mockResolver := test_mocks.NewMockDNSResolver()
	// Don't add any CNAME records, so verification will fail
	certManager.SetDNSResolver(mockResolver)

	certManager.SetLegoRunner(func(cfg *manager.Config, store interface{}, action string, certName string, domains []string, keyType string) error {
		t.Logf("Mock Lego called: action=%s, cert=%s", action, certName)

		if action != "init" {
			t.Errorf("Expected action 'init', got: %s", action)
		}

		// First run - DNS not configured
		return manager.ErrDNSSetupNeeded
	})

	ctx := context.Background()
	err = certManager.ProcessAutoMode(ctx)

	// The error may be wrapped, so use errors.Is
	if !errors.Is(err, manager.ErrDNSSetupNeeded) {
		t.Errorf("WORKFLOW VIOLATION: Expected ErrDNSSetupNeeded on first run, got: %v", err)
	}

	// WORKFLOW STEP 2: Second run - should succeed
	t.Log("WORKFLOW: Step 2 - Auto mode run after DNS configuration")

	certManager2, err := app.NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager for second run: %v", err)
	}

	// Set up mock DNS resolver with proper CNAME records (simulating user configured DNS)
	mockResolver2 := test_mocks.NewMockDNSResolver()
	mockResolver2.AddCNAMERecord("_acme-challenge.example.com", "test-uuid.acme-dns.example.com")
	mockResolver2.AddCNAMERecord("_acme-challenge.www.example.com", "test-uuid-www.acme-dns.example.com")
	certManager2.SetDNSResolver(mockResolver2)

	certManager2.SetLegoRunner(func(cfg *manager.Config, store interface{}, action string, certName string, domains []string, keyType string) error {
		// Simulate successful certificate creation
		certPath := filepath.Join(cfg.CertStoragePath, "certificates", certName+".crt")
		os.MkdirAll(filepath.Dir(certPath), 0755)
		test_helpers.CreateTestCertificate(t, certPath, domains, 90)
		return nil
	})

	err = certManager2.ProcessAutoMode(ctx)
	if err != nil {
		t.Errorf("WORKFLOW VIOLATION: Second run should succeed, got error: %v", err)
	}

	// Verify certificate was created
	certPath := filepath.Join(certStoragePath, "certificates", "web-cert.crt")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Error("WORKFLOW VIOLATION: Certificate should exist after successful second run")
	}

	t.Log("✓ Basic auto mode workflow validated successfully")
}

// TestUserWorkflow_Renewal validates renewal detection works for command-line mode
func TestUserWorkflow_Renewal_CommandLine(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test (set RUN_INTEGRATION_TESTS=1 to run)")
	}

	// Setup test environment
	tmpDir := t.TempDir()
	certStoragePath := filepath.Join(tmpDir, "certs")
	certPath := filepath.Join(certStoragePath, "certificates", "test-cert.crt")
	metaPath := filepath.Join(certStoragePath, "certificates", "test-cert.json")

	// Create config
	config := &manager.Config{
		Email:           "test@example.com",
		AcmeServer:      "https://acme-staging-v02.api.letsencrypt.org/directory",
		AcmeDnsServer:   "https://acme-dns.example.com",
		CertStoragePath: certStoragePath,
		AutoDomains: &manager.AutoDomainsConfig{
			GraceDays: 30, // Renew if less than 30 days left
		},
	}

	// Create existing certificate that expires in 20 days (needs renewal)
	os.MkdirAll(filepath.Dir(certPath), 0755)
	domains := []string{"example.com"}
	test_helpers.CreateTestCertificate(t, certPath, domains, 20)
	test_helpers.CreateTestCertificateMetadata(t, metaPath, domains)

	// Create logger
	logger := manager.NewColorfulLogger(os.Stdout, manager.LogLevelDebug, false, false)

	// Create certificate manager
	certManager, err := app.NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	renewalCalled := false
	certManager.SetLegoRunner(func(cfg *manager.Config, store interface{}, action string, certName string, domains []string, keyType string) error {
		t.Logf("Mock Lego called: action=%s", action)

		if action == "renew" {
			renewalCalled = true
			// Simulate successful renewal
			test_helpers.CreateTestCertificate(t, certPath, domains, 90)
		}

		return nil
	})

	// Process the certificate that needs renewal
	args := []string{"test-cert@example.com"}
	ctx := context.Background()
	err = certManager.ProcessManualMode(ctx, args)

	if err != nil {
		t.Fatalf("Processing failed: %v", err)
	}

	if !renewalCalled {
		t.Error("WORKFLOW VIOLATION: Certificate expiring in 20 days should trigger renewal")
	}

	t.Log("✓ Command-line renewal workflow validated successfully")
}

// TestUserWorkflow_Renewal_AutoMode validates renewal detection works for auto mode
func TestUserWorkflow_Renewal_AutoMode(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test (set RUN_INTEGRATION_TESTS=1 to run)")
	}

	// Setup test environment
	tmpDir := t.TempDir()
	certStoragePath := filepath.Join(tmpDir, "certs")

	// Create config with auto_domains
	config := &manager.Config{
		Email:           "test@example.com",
		AcmeServer:      "https://acme-staging-v02.api.letsencrypt.org/directory",
		AcmeDnsServer:   "https://acme-dns.example.com",
		CertStoragePath: certStoragePath,
		AutoDomains: &manager.AutoDomainsConfig{
			GraceDays: 30, // Renew if less than 30 days left
			Certs: map[string]manager.CertConfig{
				"web-cert": {
					Domains: []string{"example.com", "www.example.com"},
				},
				"api-cert": {
					Domains: []string{"api.example.com"},
				},
			},
		},
	}

	// Create existing certificates with different expiry times
	// web-cert: expires in 20 days (needs renewal with 30 day threshold)
	webCertPath := filepath.Join(certStoragePath, "certificates", "web-cert.crt")
	webMetaPath := filepath.Join(certStoragePath, "certificates", "web-cert.json")
	os.MkdirAll(filepath.Dir(webCertPath), 0755)
	test_helpers.CreateTestCertificate(t, webCertPath, []string{"example.com", "www.example.com"}, 20)
	test_helpers.CreateTestCertificateMetadata(t, webMetaPath, []string{"example.com", "www.example.com"})

	// api-cert: expires in 60 days (doesn't need renewal)
	apiCertPath := filepath.Join(certStoragePath, "certificates", "api-cert.crt")
	apiMetaPath := filepath.Join(certStoragePath, "certificates", "api-cert.json")
	test_helpers.CreateTestCertificate(t, apiCertPath, []string{"api.example.com"}, 60)
	test_helpers.CreateTestCertificateMetadata(t, apiMetaPath, []string{"api.example.com"})

	// Create logger
	logger := manager.NewColorfulLogger(os.Stdout, manager.LogLevelDebug, false, false)

	// Create certificate manager
	certManager, err := app.NewCertificateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}

	renewals := make(map[string]bool)
	certManager.SetLegoRunner(func(cfg *manager.Config, store interface{}, action string, certName string, domains []string, keyType string) error {
		t.Logf("Mock Lego called: action=%s, cert=%s", action, certName)

		if action == "renew" {
			renewals[certName] = true
			// Simulate successful renewal
			certPath := filepath.Join(cfg.CertStoragePath, "certificates", certName+".crt")
			test_helpers.CreateTestCertificate(t, certPath, domains, 90)
		}

		return nil
	})

	// Process auto mode
	ctx := context.Background()
	err = certManager.ProcessAutoMode(ctx)

	if err != nil {
		t.Fatalf("Auto mode processing failed: %v", err)
	}

	// Verify only web-cert was renewed (expires in 20 days with 30 day threshold)
	if !renewals["web-cert"] {
		t.Error("WORKFLOW VIOLATION: web-cert expiring in 20 days should trigger renewal (30 day threshold)")
	}

	if renewals["api-cert"] {
		t.Error("WORKFLOW VIOLATION: api-cert expiring in 60 days should NOT trigger renewal (30 day threshold)")
	}

	t.Log("✓ Auto mode renewal workflow validated successfully")
}
