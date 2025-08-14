//go:build testutils

package test_helpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
	"gopkg.in/yaml.v3"
)

// WriteTestConfig writes a test configuration file
func WriteTestConfig(path string, config *manager.Config) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// CreateTestCertificate creates a test certificate file with specified domains and expiry
func CreateTestCertificate(t *testing.T, path string, domains []string, daysValid int) {
	// Generate a self-signed certificate
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Duration(daysValid) * 24 * time.Hour),
		DNSNames:     domains,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Write certificate to file
	certOut, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create certificate file: %v", err)
	}
	defer certOut.Close()

	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err != nil {
		t.Fatalf("Failed to write certificate: %v", err)
	}

	// Also write the private key (same path but with .key extension)
	keyPath := strings.TrimSuffix(path, ".crt") + ".key"
	keyOut, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	defer keyOut.Close()

	privKeyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}

	err = pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privKeyDER})
	if err != nil {
		t.Fatalf("Failed to write private key: %v", err)
	}
}

// CreateTestCertificateMetadata creates a test certificate metadata file (Lego format)
func CreateTestCertificateMetadata(t *testing.T, path string, domains []string) {
	// Create a simple metadata structure similar to what Lego creates
	metadata := map[string]interface{}{
		"domain": domains[0], // Primary domain
		"domains": domains,
		"certificate": strings.TrimSuffix(path, ".json") + ".crt",
		"key": strings.TrimSuffix(path, ".json") + ".key",
		"issuer_certificate": "",
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to write metadata file: %v", err)
	}
}

// WriteTestAccounts writes test ACME-DNS accounts to a file
func WriteTestAccounts(t *testing.T, path string, accounts map[string]manager.AcmeDnsAccount) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	data, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal accounts: %v", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("Failed to write accounts file: %v", err)
	}
}
