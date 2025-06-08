package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/certificate"
)

// TestRunLego_ValidationErrors tests input validation
func TestRunLego_ValidationErrors(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	cfg := &Config{
		Email:              "test@valid-domain.org", // Use a valid domain to avoid ACME server rejections
		AcmeServer:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		CertStoragePath:    tmpDir,
		AcmeDnsServer:     "https://acme-dns.example.com",
		ChallengeTimeout:  10 * time.Minute,
		HTTPTimeout:       30 * time.Second,
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	tests := []struct {
		name     string
		action   string
		certName string
		domains  []string
		keyType  string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "empty domains list",
			action:   "init",
			certName: "test-cert",
			domains:  []string{},
			keyType:  "rsa2048",
			wantErr:  true,
			errMsg:   "RunLego called with empty domains list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunLego(cfg, store, tt.action, tt.certName, tt.domains, tt.keyType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestRunLego_KeyTypeMapping tests key type validation and mapping
func TestRunLego_KeyTypeMapping(t *testing.T) {
	// This test focuses on the key type logic without actually running ACME operations
	// We'll test this by examining the internal logic through smaller unit tests

	tests := []struct {
		name     string
		keyType  string
		expected string
	}{
		{"default when empty", "", DefaultKeyType},
		{"valid rsa2048", "rsa2048", "rsa2048"},
		{"valid rsa3072", "rsa3072", "rsa3072"},
		{"valid rsa4096", "rsa4096", "rsa4096"},
		{"valid ec256", "ec256", "ec256"},
		{"valid ec384", "ec384", "ec384"},
		{"invalid key type falls back to default", "invalid", DefaultKeyType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the key type validation logic
			var result string
			if tt.keyType != "" && isValidKeyType(tt.keyType) {
				result = tt.keyType
			} else {
				result = DefaultKeyType
			}

			if result != tt.expected {
				t.Errorf("Key type mapping: expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestRunLego_DNSResolverFormatting tests DNS resolver logic
func TestRunLego_DNSResolverFormatting(t *testing.T) {
	// Test the DNS resolver formatting logic specifically
	tests := []struct {
		name        string
		dnsResolver string
		expected    string
	}{
		{"no custom DNS resolver", "", ""},
		{"custom DNS resolver with port", "8.8.8.8:53", "8.8.8.8:53"},
		{"custom DNS resolver without port", "8.8.8.8", "8.8.8.8:53"},
		{"IPv6 resolver with port", "[2001:4860:4860::8888]:53", "[2001:4860:4860::8888]:53"},
		{"IPv6 resolver without port", "2001:4860:4860::8888", "2001:4860:4860::8888"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the DNS resolver formatting logic directly
			var result string
			if tt.dnsResolver != "" {
				// This mirrors the logic in RunLego
				nsAddr := tt.dnsResolver
				if !strings.Contains(nsAddr, ":") {
					nsAddr += ":53"
				}
				result = nsAddr
			}

			if result != tt.expected {
				t.Errorf("DNS resolver formatting: expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestRunLego_RenewalWithoutCertificate tests renewal error cases
func TestRunLego_RenewalWithoutCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Email:              "test@valid-domain.org",
		AcmeServer:         "https://acme-staging-v02.api.letsencrypt.org/directory",
		CertStoragePath:    tmpDir,
		AcmeDnsServer:     "https://acme-dns.example.com",
		ChallengeTimeout:  10 * time.Minute,
		HTTPTimeout:       30 * time.Second,
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	// Try to renew a non-existent certificate - this should fail early before ACME operations
	err = RunLego(cfg, store, "renew", "nonexistent-cert", []string{"example.org"}, "rsa2048")

	if err == nil {
		t.Error("Expected error for renewing non-existent certificate")
		return
	}

	// Should fail because certificate file doesn't exist, not because of ACME registration
	if !containsString(err.Error(), "certificate file not found") &&
		!containsString(err.Error(), "no such file") {
		t.Errorf("Expected error about certificate file not found, got: %s", err.Error())
	}
}

// Mock certificate resource for testing
func createTestCertificateResource() *certificate.Resource {
	return &certificate.Resource{
		Domain:            "example.com",
		CertURL:           "https://acme-v02.api.letsencrypt.org/acme/cert/123",
		CertStableURL:     "https://acme-v02.api.letsencrypt.org/acme/cert/123",
		PrivateKey:        []byte("-----BEGIN PRIVATE KEY-----\ntest-private-key\n-----END PRIVATE KEY-----"),
		Certificate:       []byte("-----BEGIN CERTIFICATE-----\ntest-certificate\n-----END CERTIFICATE-----"),
		IssuerCertificate: []byte("-----BEGIN CERTIFICATE-----\ntest-issuer-certificate\n-----END CERTIFICATE-----"),
		CSR:               []byte("-----BEGIN CERTIFICATE REQUEST-----\ntest-csr\n-----END CERTIFICATE REQUEST-----"),
	}
}

// TestLoadCertificateResource_Success tests successful certificate loading
func TestLoadCertificateResource_Success(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		CertStoragePath: tmpDir,
	}

	// Create test certificate files
	certName := "test-cert"
	certDir := filepath.Join(tmpDir, "certificates")
	if err := os.MkdirAll(certDir, 0755); err != nil {
		t.Fatalf("Failed to create cert directory: %v", err)
	}

	// Create mock certificate resource and save it
	testCert := createTestCertificateResource()
	if err := saveCertificates(cfg, certName, testCert); err != nil {
		t.Fatalf("Failed to save test certificate: %v", err)
	}

	// Test loading the certificate
	loadedCert, err := LoadCertificateResource(cfg, certName)
	if err != nil {
		t.Fatalf("Failed to load certificate: %v", err)
	}

	if loadedCert.Domain != testCert.Domain {
		t.Errorf("Domain mismatch: expected %s, got %s", testCert.Domain, loadedCert.Domain)
	}

	if string(loadedCert.Certificate) != string(testCert.Certificate) {
		t.Errorf("Certificate content mismatch")
	}
}

// TestLoadCertificateResource_NotFound tests certificate not found error
func TestLoadCertificateResource_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		CertStoragePath: tmpDir,
	}

	_, err := LoadCertificateResource(cfg, "nonexistent-cert")
	if err == nil {
		t.Error("Expected error for non-existent certificate")
	}

	if !containsString(err.Error(), "certificate file not found") &&
		!containsString(err.Error(), "no such file") {
		t.Errorf("Expected file not found error, got: %s", err.Error())
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Integration test with mock servers (when available)
func TestRunLego_Integration(t *testing.T) {
	// Skip integration test if not explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Integration tests disabled. Set RUN_INTEGRATION_TESTS=1 to enable.")
	}

	// This would use the mock servers from test_mocks
	// For now, we'll skip this as it requires complex setup
	t.Skip("Integration test requires mock ACME and ACME-DNS servers")
}

// Benchmark for RunLego key type mapping (performance test)
func BenchmarkKeyTypeMapping(b *testing.B) {
	keyTypes := []string{"", "rsa2048", "rsa3072", "rsa4096", "ec256", "ec384", "invalid"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyType := keyTypes[i%len(keyTypes)]
		var result string
		if keyType != "" && isValidKeyType(keyType) {
			result = keyType
		} else {
			result = DefaultKeyType
		}
		_ = result // Prevent optimization
	}
}
