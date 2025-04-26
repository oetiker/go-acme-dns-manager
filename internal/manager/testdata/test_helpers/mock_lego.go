package test_helpers

import (
	"os"
	"path/filepath"

	"github.com/oetiker/go-acme-dns-manager/internal/manager"
)

// MockLegoRun is a mock implementation of RunLego
// It simulates the creation of certificates but without making any real ACME calls
func MockLegoRun(cfg *manager.Config, store *manager.AccountStore, action string, certName string, domains []string) error {
	// Create certificate directories
	certsDir := filepath.Join(cfg.CertStoragePath, "certificates")
	if err := os.MkdirAll(certsDir, manager.DirPermissions); err != nil {
		return err
	}

	// Generate mock certificate files
	files := []struct {
		path        string
		content     string
		permissions os.FileMode
	}{
		{
			path:        filepath.Join(certsDir, certName+".crt"),
			content:     "-----BEGIN CERTIFICATE-----\nMOCK CERTIFICATE FOR TESTING\n-----END CERTIFICATE-----",
			permissions: manager.CertificatePermissions,
		},
		{
			path:        filepath.Join(certsDir, certName+".key"),
			content:     "-----BEGIN PRIVATE KEY-----\nMOCK PRIVATE KEY FOR TESTING\n-----END PRIVATE KEY-----",
			permissions: manager.PrivateKeyPermissions,
		},
		{
			path:        filepath.Join(certsDir, certName+".json"),
			content:     `{"domain":"` + domains[0] + `","domains":["` + domains[0] + `"],"certificate":"MOCK CERT DATA","key":"MOCK KEY DATA"}`,
			permissions: manager.PrivateKeyPermissions,
		},
		{
			path:        filepath.Join(certsDir, certName+".issuer.crt"),
			content:     "-----BEGIN CERTIFICATE-----\nMOCK ISSUER CERTIFICATE FOR TESTING\n-----END CERTIFICATE-----",
			permissions: manager.CertificatePermissions,
		},
	}

	for _, file := range files {
		if err := os.WriteFile(file.path, []byte(file.content), file.permissions); err != nil {
			return err
		}
	}

	return nil
}
