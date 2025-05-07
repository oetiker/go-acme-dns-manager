package manager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "valid config",
			config: `
email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
acme_dns_server: "https://auth.acme-dns.io"
cert_storage_path: "./data"
key_type: "ec256"
`,
			wantErr: false,
		},
		{
			name: "unknown field",
			config: `
email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
acme_dns_server: "https://auth.acme-dns.io"
cert_storage_path: "./data"
unknown_field: "should fail"
`,
			wantErr: true,
		},
		{
			name: "invalid key_type",
			config: `
email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
acme_dns_server: "https://auth.acme-dns.io"
cert_storage_path: "./data"
key_type: "invalid_key_type"
`,
			wantErr: true,
		},
		{
			name: "valid auto domains config",
			config: `
email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
acme_dns_server: "https://auth.acme-dns.io"
cert_storage_path: "./data"
auto_domains:
  grace_days: 30
  certs:
    my-cert:
      domains:
        - example.com
        - www.example.com
      key_type: "ec256"
`,
			wantErr: false,
		},
		{
			name: "invalid auto domains config",
			config: `
email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
acme_dns_server: "https://auth.acme-dns.io"
cert_storage_path: "./data"
auto_domains:
  grace_days: 30
  certs:
    my-cert:
      domains:
        - example.com
      key_type: "invalid_key_type"
      unknown_field: "should fail"
`,
			wantErr: true,
		},
		{
			name: "missing required field",
			config: `
email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
cert_storage_path: "./data"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0600); err != nil {
				t.Fatalf("Failed to write test config file: %v", err)
			}

			// Test LoadConfig function, which calls validateConfig internally
			_, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				if err != nil {
					t.Logf("Error details: %v", err)
				}
			}

			// Also test validateConfig function directly
			err = validateConfig([]byte(tt.config))
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() direct call error = %v, wantErr %v", err, tt.wantErr)
				if err != nil {
					t.Logf("Error details: %v", err)
				}
			} else if err != nil {
				// Log the error for debugging when it's expected
				t.Logf("Got expected error from validateConfig: %v", err)
			}
		})
	}
}
