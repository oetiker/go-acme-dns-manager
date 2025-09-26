//go:build testutils
// +build testutils

package test_helpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// Constants for permissions
const (
	DirPermissions         = 0755
	CertificatePermissions = 0644
	PrivateKeyPermissions  = 0600
)

// MockLegoRun is a mock implementation of RunLego
// It simulates the creation of certificates but creates real X.509 certificates with all requested domains
func MockLegoRun(cfg *manager.Config, store interface{}, action string, certName string, domains []string, keyType string) error {
	// Create certificate directories
	certsDir := filepath.Join(cfg.CertStoragePath, "certificates")
	if err := os.MkdirAll(certsDir, DirPermissions); err != nil {
		return err
	}

	// Generate a real X.509 certificate with all requested domains
	certPath := filepath.Join(certsDir, certName+".crt")
	keyPath := filepath.Join(certsDir, certName+".key")

	err := createMockCertificate(certPath, keyPath, domains, 90) // 90 days validity
	if err != nil {
		return err
	}

	// Generate remaining mock files (issuer cert and metadata)
	files := []struct {
		path        string
		content     string
		permissions os.FileMode
	}{
		{
			path:        filepath.Join(certsDir, certName+".json"),
			content:     `{"domain":"` + domains[0] + `","certificate":"MOCK CERT DATA","key":"MOCK KEY DATA"}`,
			permissions: PrivateKeyPermissions,
		},
		{
			path:        filepath.Join(certsDir, certName+".issuer.crt"),
			content:     "-----BEGIN CERTIFICATE-----\nMOCK ISSUER CERTIFICATE FOR TESTING\n-----END CERTIFICATE-----",
			permissions: CertificatePermissions,
		},
	}

	for _, file := range files {
		if err := os.WriteFile(file.path, []byte(file.content), file.permissions); err != nil {
			return err
		}
	}

	return nil
}

// createMockCertificate creates a real X.509 certificate with all specified domains
func createMockCertificate(certPath, keyPath string, domains []string, daysUntilExpiry int) error {
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

	// Set ALL the DNS names - this is the critical fix
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
	os.Chmod(keyPath, PrivateKeyPermissions)

	return nil
}
