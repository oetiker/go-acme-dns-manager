package app

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// testConfigLogger implements LoggerInterface for testing config changes
type testConfigLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
}

func (m *testConfigLogger) Debug(msg string, args ...interface{})             { m.debugMessages = append(m.debugMessages, fmt.Sprintf(msg, args...)) }
func (m *testConfigLogger) Info(msg string, args ...interface{})              { m.infoMessages = append(m.infoMessages, fmt.Sprintf(msg, args...)) }
func (m *testConfigLogger) Warn(msg string, args ...interface{})              { m.warnMessages = append(m.warnMessages, fmt.Sprintf(msg, args...)) }
func (m *testConfigLogger) Error(msg string, args ...interface{})             { m.errorMessages = append(m.errorMessages, fmt.Sprintf(msg, args...)) }
func (m *testConfigLogger) Debugf(format string, args ...interface{})         { m.debugMessages = append(m.debugMessages, fmt.Sprintf(format, args...)) }
func (m *testConfigLogger) Infof(format string, args ...interface{})          { m.infoMessages = append(m.infoMessages, fmt.Sprintf(format, args...)) }
func (m *testConfigLogger) Warnf(format string, args ...interface{})          { m.warnMessages = append(m.warnMessages, fmt.Sprintf(format, args...)) }
func (m *testConfigLogger) Errorf(format string, args ...interface{})         { m.errorMessages = append(m.errorMessages, fmt.Sprintf(format, args...)) }
func (m *testConfigLogger) Importantf(format string, args ...interface{})     { m.infoMessages = append(m.infoMessages, fmt.Sprintf(format, args...)) }

// TestCertificateConfigChange tests that certificates are renewed when config changes
// to request additional domains, even if the certificate hasn't expired yet
func TestCertificateConfigChange(t *testing.T) {
	// Create a temporary directory for test
	tempDir, err := os.MkdirTemp("", "cert-config-change-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Create certificates directory
	certsDir := filepath.Join(tempDir, "certificates")
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		t.Fatalf("Failed to create certificates directory: %v", err)
	}

	// Test logger
	logger := &testConfigLogger{}

	// Create test certificate with only example.com (expires in 60 days)
	certPath := filepath.Join(certsDir, "example.com.crt")
	err = createValidCertificate(certPath, []string{"example.com"}, 60)
	if err != nil {
		t.Fatalf("Failed to create test certificate: %v", err)
	}

	// Create metadata file to simulate existing certificate
	metadataPath := filepath.Join(certsDir, "example.com.json")
	metadata := `{"domain":"example.com","domains":["example.com"],"certificate":"CERT","key":"KEY"}`
	if err := os.WriteFile(metadataPath, []byte(metadata), 0600); err != nil {
		t.Fatalf("Failed to create metadata file: %v", err)
	}

	// Create config requesting example.com AND www.example.com
	config := &manager.Config{
		Email:           "test@example.com",
		AcmeServer:      "https://acme-staging-v02.api.letsencrypt.org/directory",
		AcmeDnsServer:   "https://acme-dns.example.com",
		CertStoragePath: tempDir,
		AutoDomains: &manager.AutoDomainsConfig{
			GraceDays: 30, // 30 days before expiry
		},
	}

	// Create certificate manager
	cm := &CertificateManager{
		config:     config,
		logger:     logger,
		legoRunner: mockConfigChangeLegoRunner,
	}

	// Test scenario: User originally had just example.com, now wants www.example.com too
	req := CertRequest{
		Name:    "example.com",
		Domains: []string{"example.com", "www.example.com"}, // Now requesting 2 domains
		KeyType: "ec256",
	}

	// Determine what action should be taken - use 30 days for threshold
	action, err := cm.determineAction(req, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("determineAction failed: %v", err)
	}

	// The certificate is valid for 60 more days (well beyond the 30-day threshold)
	// BUT it's missing www.example.com
	// So it SHOULD return "renew" action
	if action != "renew" {
		t.Errorf("Expected action='renew' for certificate missing requested domain, got action=%s", action)
		t.Errorf("Certificate has: [example.com]")
		t.Errorf("Config requests: [example.com, www.example.com]")
		t.Errorf("Missing domain: www.example.com")
		t.Errorf("Even though certificate expires in 60 days, it should renew due to missing domain")
	}
}

// mockConfigChangeLegoRunner is a mock implementation for testing config changes
func mockConfigChangeLegoRunner(cfg *manager.Config, store interface{}, action string, certName string, domains []string, keyType string) error {
	// Mock successful renewal
	return nil
}

// createValidCertificate creates a test certificate with specified domains and expiry
func createValidCertificate(certPath string, domains []string, daysUntilExpiry int) error {
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
	defer func() { _ = certOut.Close() }()

	// Write the certificate in PEM format
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err != nil {
		return err
	}

	return nil
}

// TestCertificateConfigChangeScenarios tests various config change scenarios
func TestCertificateConfigChangeScenarios(t *testing.T) {
	testCases := []struct {
		name             string
		certDomains      []string      // Domains in existing certificate
		requestedDomains []string      // Domains requested in config
		daysUntilExpiry  int           // How many days until cert expires
		renewalDays      int           // Renewal threshold in days
		expectedAction   string        // Expected action: "renew" or "skip"
		expectedReason   string        // Why this action was taken
	}{
		{
			name:             "Config adds new domain - should renew",
			certDomains:      []string{"example.com"},
			requestedDomains: []string{"example.com", "www.example.com"},
			daysUntilExpiry:  60,
			renewalDays:      30,
			expectedAction:   "renew",
			expectedReason:   "certificate missing domains: [www.example.com]",
		},
		{
			name:             "Config adds multiple domains - should renew",
			certDomains:      []string{"example.com"},
			requestedDomains: []string{"example.com", "www.example.com", "api.example.com"},
			daysUntilExpiry:  60,
			renewalDays:      30,
			expectedAction:   "renew",
			expectedReason:   "certificate missing domains: [www.example.com api.example.com]",
		},
		{
			name:             "Config removes domain (cert has extra) - should NOT renew",
			certDomains:      []string{"example.com", "www.example.com", "old.example.com"},
			requestedDomains: []string{"example.com", "www.example.com"},
			daysUntilExpiry:  60,
			renewalDays:      30,
			expectedAction:   "skip",
			expectedReason:   "certificate is valid",
		},
		{
			name:             "Config unchanged - should NOT renew",
			certDomains:      []string{"example.com", "www.example.com"},
			requestedDomains: []string{"example.com", "www.example.com"},
			daysUntilExpiry:  60,
			renewalDays:      30,
			expectedAction:   "skip",
			expectedReason:   "certificate is valid",
		},
		{
			name:             "Near expiry but domains match - should renew",
			certDomains:      []string{"example.com", "www.example.com"},
			requestedDomains: []string{"example.com", "www.example.com"},
			daysUntilExpiry:  20, // Less than 30-day threshold
			renewalDays:      30,
			expectedAction:   "renew",
			expectedReason:   "certificate expires in",
		},
		{
			name:             "Near expiry AND missing domain - should renew",
			certDomains:      []string{"example.com"},
			requestedDomains: []string{"example.com", "www.example.com"},
			daysUntilExpiry:  20,
			renewalDays:      30,
			expectedAction:   "renew",
			expectedReason:   "certificate expires in", // Expiry is checked first
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for this test
			tempDir, err := os.MkdirTemp("", "cert-scenario-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

			// Create certificates directory
			certsDir := filepath.Join(tempDir, "certificates")
			if err := os.MkdirAll(certsDir, 0755); err != nil {
				t.Fatalf("Failed to create certificates directory: %v", err)
			}

			// Create test certificate
			certPath := filepath.Join(certsDir, "test.crt")
			err = createValidCertificate(certPath, tc.certDomains, tc.daysUntilExpiry)
			if err != nil {
				t.Fatalf("Failed to create test certificate: %v", err)
			}

			// Check if certificate needs renewal
			renewalThreshold := time.Duration(tc.renewalDays) * 24 * time.Hour
			needsRenewal, reason, _ := manager.CertificateNeedsRenewal(certPath, tc.requestedDomains, renewalThreshold)

			// Determine expected needsRenewal based on expectedAction
			expectedNeedsRenewal := (tc.expectedAction == "renew")

			if needsRenewal != expectedNeedsRenewal {
				t.Errorf("Expected needsRenewal=%v but got %v", expectedNeedsRenewal, needsRenewal)
				t.Errorf("Certificate domains: %v", tc.certDomains)
				t.Errorf("Requested domains: %v", tc.requestedDomains)
				t.Errorf("Days until expiry: %d, renewal threshold: %d days", tc.daysUntilExpiry, tc.renewalDays)
				if reason != "" {
					t.Errorf("Reason given: %s", reason)
				}
			}

			// Check if reason contains expected text
			if tc.expectedReason != "" && needsRenewal {
				// For expiry reasons, just check prefix
				if tc.expectedReason == "certificate expires in" {
					if len(reason) < len(tc.expectedReason) || reason[:len(tc.expectedReason)] != tc.expectedReason {
						t.Errorf("Expected reason starting with %q but got %q", tc.expectedReason, reason)
					}
				} else if reason != tc.expectedReason {
					t.Errorf("Expected reason %q but got %q", tc.expectedReason, reason)
				}
			}

			// If we expect skip but got renewal, show why
			if tc.expectedAction == "skip" && needsRenewal {
				t.Errorf("Certificate should NOT need renewal but got: %s", reason)
			}
		})
	}
}
