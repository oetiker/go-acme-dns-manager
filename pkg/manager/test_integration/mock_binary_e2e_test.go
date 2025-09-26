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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// TestMockBinaryDomainChange tests the complete application using the mock binary
// to verify that domain changes trigger certificate renewal
func TestMockBinaryDomainChange(t *testing.T) {
	// Skip unless explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping mock binary E2E test. Set RUN_INTEGRATION_TESTS=1 to enable.")
	}

	// Create a temporary directory for test data
	tempDir, err := os.MkdirTemp("", "mock-binary-e2e-test")
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
	err = createMockE2ECertificate(certPath, keyPath, []string{"example.com"}, 60)
	if err != nil {
		t.Fatalf("Failed to create test certificate: %v", err)
	}

	// Create metadata file to simulate existing certificate
	metadata := `{"domain":"example.com","domains":["example.com"],"certificate":"CERT","key":"KEY"}`
	if err := os.WriteFile(metadataPath, []byte(metadata), 0600); err != nil {
		t.Fatalf("Failed to create metadata file: %v", err)
	}

	// Step 2: Create config that requests both example.com AND www.example.com
	t.Log("Step 2: Creating config that requests both example.com and www.example.com")
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
acme_dns_server: "https://acme-dns.example.com"
cert_storage_path: "` + tempDir + `"

# Auto domains configuration - requesting TWO domains
auto_domains:
  grace_days: 30  # Renew 30 days before expiry
  certs:
    test.example.com:
      domains:
        - example.com
        - www.example.com  # NEW domain added to config
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Step 3: Verify the mock binary exists (should be built by make target)
	t.Log("Step 3: Checking for mock binary")
	// Get absolute path to project root
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(wd, "../../..")
	mockBinaryPath := filepath.Join(projectRoot, "build", "go-acme-dns-manager-mock")
	if _, err := os.Stat(mockBinaryPath); os.IsNotExist(err) {
		t.Fatalf("Mock binary not found at %s. Run 'make build-mock' first.", mockBinaryPath)
	}

	// Step 4: Run the mock binary in auto mode
	t.Log("Step 4: Running mock binary to test domain change detection")

	cmd := exec.Command(mockBinaryPath, "-auto", "-config", configPath, "-debug")
	cmd.Dir = projectRoot // Run from project root

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	t.Logf("Mock binary output:\n%s", outputStr)

	if err != nil {
		// Even if there's an error, check if it detected the domain mismatch
		t.Logf("Mock binary exited with error (this might be expected): %v", err)
	}

	// Step 5: Verify domain change detection
	t.Log("Step 5: Verifying that domain change was detected")

	// Check for various indicators that renewal was triggered due to domain mismatch
	domainMismatchDetected := false
	renewalTriggered := false

	// Look for specific domain mismatch messages
	if strings.Contains(outputStr, "certificate missing domains: [www.example.com]") {
		domainMismatchDetected = true
		t.Log("✅ SUCCESS: Found exact domain mismatch message")
	}
	if strings.Contains(outputStr, "missing domains") {
		domainMismatchDetected = true
		t.Log("✅ SUCCESS: Found general missing domains message")
	}
	if strings.Contains(outputStr, "needs renewal") {
		renewalTriggered = true
		t.Log("✅ SUCCESS: Certificate renewal was triggered")
	}
	if strings.Contains(outputStr, "MockLegoRun called with action=renew") {
		renewalTriggered = true
		t.Log("✅ SUCCESS: Mock Lego runner was called for renewal")
	}
	if strings.Contains(outputStr, "Renewing certificate") {
		renewalTriggered = true
		t.Log("✅ SUCCESS: Certificate renewal process started")
	}

	// Verify that mock infrastructure was properly initialized
	mockInfrastructureWorking := false
	if strings.Contains(outputStr, "Mock ACME-DNS server running") ||
		strings.Contains(outputStr, "Mock ACME server running") ||
		strings.Contains(outputStr, "🧪") {
		mockInfrastructureWorking = true
		t.Log("✅ SUCCESS: Mock infrastructure is working")
	}

	// Report results
	if !mockInfrastructureWorking {
		t.Error("❌ FAILED: Mock infrastructure not properly initialized")
	}

	if !domainMismatchDetected {
		t.Error("❌ FAILED: Domain mismatch was NOT detected")
		t.Error("The certificate has: [example.com]")
		t.Error("The config requests: [example.com, www.example.com]")
		t.Error("Expected to see 'certificate missing domains: [www.example.com]'")
	}

	if !renewalTriggered {
		t.Error("❌ FAILED: Certificate renewal was NOT triggered")
		t.Error("Even though the certificate is missing www.example.com")
	}

	if domainMismatchDetected && renewalTriggered {
		t.Log("🎉 SUCCESS: Mock binary correctly detected domain changes and triggered renewal!")
	}

	// Additional checks
	if strings.Contains(outputStr, "no renewal needed") {
		t.Error("❌ UNEXPECTED: Mock binary incorrectly said no renewal needed")
	}
}

// createMockE2ECertificate creates a test certificate and private key for E2E testing
func createMockE2ECertificate(certPath, keyPath string, domains []string, daysUntilExpiry int) error {
	// Generate a private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Mock Test Organization"},
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
