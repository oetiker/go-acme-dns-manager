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

// TestRealEndToEndConfigChange tests the ACTUAL application binary
// to verify that config changes trigger certificate renewal
func TestRealEndToEndConfigChange(t *testing.T) {
	// Skip unless explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping real E2E config change test. Set RUN_INTEGRATION_TESTS=1 to enable.")
	}

	// Create a temporary directory for test data
	tempDir, err := os.MkdirTemp("", "real-e2e-config-test")
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
	err = createRealE2ECertificate(certPath, keyPath, []string{"example.com"}, 60)
	if err != nil {
		t.Fatalf("Failed to create test certificate: %v", err)
	}

	// Create metadata file to simulate existing certificate
	metadata := `{"domain":"example.com","domains":["example.com"],"certificate":"CERT","key":"KEY"}`
	if err := os.WriteFile(metadataPath, []byte(metadata), 0600); err != nil {
		t.Fatalf("Failed to create metadata file: %v", err)
	}

	// Create ACME DNS accounts file
	accountsPath := filepath.Join(tempDir, "acme-dns-accounts.json")
	accounts := `{
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
	if err := os.WriteFile(accountsPath, []byte(accounts), 0600); err != nil {
		t.Fatalf("Failed to create accounts file: %v", err)
	}

	// Step 2: Create config that requests both example.com AND www.example.com
	t.Log("Step 2: Creating config that requests both example.com and www.example.com")
	configPath := filepath.Join(tempDir, "config.yaml")

	// First, let's create a simple test to check what the app does in auto mode
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

	// Step 3: Build the application binary
	t.Log("Step 3: Building the application binary")
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(tempDir, "go-acme-dns-manager"), "../../../cmd/go-acme-dns-manager")
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build application: %v\nOutput: %s", err, buildOutput)
	}

	// Step 4: Run the application in auto mode with dry-run or test mode
	t.Log("Step 4: Running application in auto mode to check if it detects the need for renewal")

	// We'll run the app and check its output to see if it tries to renew
	appPath := filepath.Join(tempDir, "go-acme-dns-manager")

	// First, let's see what the app does in debug mode
	cmd := exec.Command(appPath, "-auto", "-config", configPath, "-debug")

	// Set environment to ensure we don't actually hit real ACME servers
	cmd.Env = append(os.Environ(),
		"LEGO_DISABLE_CNAME_SUPPORT=true",  // Disable CNAME checks
		"TEST_MODE=true",                   // If the app respects this
	)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	t.Logf("Application output:\n%s", outputStr)

	// Step 5: Analyze the output to see if renewal was attempted
	t.Log("Step 5: Analyzing output to check if renewal was triggered")

	// Check for various indicators that renewal was triggered
	renewalTriggered := false
	reasonFound := ""

	// Look for indicators in the output
	if strings.Contains(outputStr, "needs renewal") {
		renewalTriggered = true
		reasonFound = "Found 'needs renewal' in output"
	}
	if strings.Contains(outputStr, "missing domains") {
		renewalTriggered = true
		reasonFound = "Found 'missing domains' in output"
	}
	if strings.Contains(outputStr, "Certificate missing domains: [www.example.com]") {
		renewalTriggered = true
		reasonFound = "Found specific missing domain message"
	}
	if strings.Contains(outputStr, "Renewing certificate for test.example.com") {
		renewalTriggered = true
		reasonFound = "Found renewal action message"
	}
	if strings.Contains(outputStr, "action renew") {
		renewalTriggered = true
		reasonFound = "Found 'action renew' in output"
	}

	// Also check if it says the certificate is valid and no renewal needed
	if strings.Contains(outputStr, "Certificate for test.example.com is valid") &&
	   strings.Contains(outputStr, "no renewal needed") {
		renewalTriggered = false
		reasonFound = "Certificate reported as valid with no renewal needed"
	}

	// Report results
	if !renewalTriggered {
		t.Error("FAILED: Application did NOT trigger certificate renewal")
		t.Error("Even though the certificate is missing www.example.com")
		t.Error("Certificate has: [example.com]")
		t.Error("Config requests: [example.com, www.example.com]")
		t.Errorf("Reason: %s", reasonFound)
		t.Log("This confirms the bug: domain changes in config don't trigger renewal")
	} else {
		t.Logf("SUCCESS: Certificate renewal was triggered")
		t.Logf("Reason detected: %s", reasonFound)
	}

	// Additional check: See if the certificate was only checked for expiry
	if strings.Contains(outputStr, "Certificate for test.example.com is valid") {
		t.Log("WARNING: App might only be checking expiry, not domain list")
	}
}

// createRealE2ECertificate creates a test certificate and private key
func createRealE2ECertificate(certPath, keyPath string, domains []string, daysUntilExpiry int) error {
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
