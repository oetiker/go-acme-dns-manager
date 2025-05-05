package main

import (
	"os"
	"strings"
	"testing"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// No need to redefine parseCertArg - using the one from main.go

func TestParseCertArg(t *testing.T) {
	testCases := []struct {
		name          string
		arg           string
		wantCertName  string
		wantDomains   []string
		wantKeyType   string
		wantErr       bool
		wantErrPrefix string
	}{
		{
			name:         "Valid input with multiple domains",
			arg:          "mycert@example.com,www.example.com",
			wantCertName: "mycert",
			wantDomains:  []string{"example.com", "www.example.com"},
			wantKeyType:  "",
			wantErr:      false,
		},
		{
			name:         "Valid input with single domain",
			arg:          "mycert@example.com",
			wantCertName: "mycert",
			wantDomains:  []string{"example.com"},
			wantKeyType:  "",
			wantErr:      false,
		},
		{
			name:         "Simple domain format (shorthand)",
			arg:          "example.com",
			wantCertName: "example.com",
			wantDomains:  []string{"example.com"},
			wantKeyType:  "",
			wantErr:      false,
		},
		{
			name:          "Empty domains list",
			arg:           "mycert@",
			wantCertName:  "",
			wantDomains:   nil,
			wantKeyType:   "",
			wantErr:       true,
			wantErrPrefix: "invalid format:",
		},
		{
			name:          "Empty cert name",
			arg:           "@example.com",
			wantCertName:  "",
			wantDomains:   nil,
			wantKeyType:   "",
			wantErr:       true,
			wantErrPrefix: "invalid format:",
		},
		{
			name:          "Invalid parameter format",
			arg:           "my/cert@example.com",
			wantCertName:  "my",
			wantDomains:   nil,
			wantKeyType:   "",
			wantErr:       true,
			wantErrPrefix: "invalid format:",
		},
		{
			name:         "Whitespace trimming in domains",
			arg:          "mycert@example.com, www.example.com ",
			wantCertName: "mycert",
			wantDomains:  []string{"example.com", "www.example.com"},
			wantKeyType:  "",
			wantErr:      false,
		},
		{
			name:         "With key_type parameter",
			arg:          "mycert@example.com/key_type=ec384",
			wantCertName: "mycert",
			wantDomains:  []string{"example.com"},
			wantKeyType:  "ec384",
			wantErr:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			certName, domains, keyType, err := parseCertArg(tc.arg)

			// Check error expectation
			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tc.wantErrPrefix != "" && !strings.HasPrefix(err.Error(), tc.wantErrPrefix) {
					t.Errorf("Error doesn't match expected prefix: got %v, want prefix %s", err, tc.wantErrPrefix)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check key type if expected
			if tc.wantKeyType != keyType {
				t.Errorf("keyType = %v, want %v", keyType, tc.wantKeyType)
			}

			// Check cert name
			if certName != tc.wantCertName {
				t.Errorf("certName = %v, want %v", certName, tc.wantCertName)
			}

			// Check domains
			if len(domains) != len(tc.wantDomains) {
				t.Errorf("len(domains) = %v, want %v", len(domains), len(tc.wantDomains))
				return
			}
			for i, domain := range domains {
				if domain != tc.wantDomains[i] {
					t.Errorf("domains[%d] = %v, want %v", i, domain, tc.wantDomains[i])
				}
			}
		})
	}
}

// Test specifically for domain name validation in parseCertArg
func TestParseCertArgDomainValidation(t *testing.T) {
	testCases := []struct {
		name          string
		arg           string
		wantErr       bool
		wantErrPrefix string
	}{
		// Valid domain cases
		{
			name:    "Valid domain",
			arg:     "example.com",
			wantErr: false,
		},
		{
			name:    "Valid subdomain",
			arg:     "sub.example.com",
			wantErr: false,
		},
		{
			name:    "Valid wildcard domain",
			arg:     "*.example.com",
			wantErr: false,
		},
		{
			name:    "Valid domain with multiple labels",
			arg:     "a.b.c.example.com",
			wantErr: false,
		},
		{
			name:    "Valid domain with cert name",
			arg:     "mycert@example.com",
			wantErr: false,
		},
		{
			name:    "Valid domain with numbers",
			arg:     "example123.com",
			wantErr: false,
		},
		{
			name:    "Valid domain with hyphen",
			arg:     "my-domain.com",
			wantErr: false,
		},
		{
			name:    "Valid multiple domains",
			arg:     "mycert@example.com,sub.example.com,*.example.com",
			wantErr: false,
		},

		// Invalid domain cases
		{
			name:          "Invalid domain with underscore",
			arg:           "invalid_domain.com",
			wantErr:       true,
			wantErrPrefix: "invalid domain name",
		},
		{
			name:          "Invalid domain starts with hyphen",
			arg:           "-invalid.com",
			wantErr:       true,
			wantErrPrefix: "invalid domain name",
		},
		{
			name:          "Invalid domain ends with hyphen",
			arg:           "invalid-.com",
			wantErr:       true,
			wantErrPrefix: "invalid domain name",
		},
		{
			name:          "Invalid domain single label",
			arg:           "localhost",
			wantErr:       true,
			wantErrPrefix: "invalid domain name",
		},
		{
			name:          "Double wildcard",
			arg:           "*.*.example.com",
			wantErr:       true,
			wantErrPrefix: "invalid domain name",
		},
		{
			name:          "Invalid with multiple domains",
			arg:           "mycert@example.com,invalid_domain.com",
			wantErr:       true,
			wantErrPrefix: "invalid domain name",
		},
		{
			name:          "Invalid subdomain with space",
			arg:           "sub domain.example.com",
			wantErr:       true,
			wantErrPrefix: "invalid domain name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := parseCertArg(tc.arg)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tc.wantErrPrefix != "" && !strings.HasPrefix(err.Error(), tc.wantErrPrefix) {
					t.Errorf("Error doesn't match expected prefix: got %v, want prefix %s", err, tc.wantErrPrefix)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// An integration test that can be run if environment variables are set
func TestMainIntegration(t *testing.T) {
	// Skip this test by default - only run in CI with specific flags
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to enable")
	}

	// Save original args and restore them after the test
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Set up a test config path
	tempFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("Warning: Failed to remove temporary file: %v", err)
		}
	}()

	// Write a test config with staging ACME server
	configContent := `
email: "test@example.com"
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory"
key_type: "ec256"
acme_dns_server: "https://acme-dns.example.com"
cert_storage_path: ".lego-test"
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temporary file: %v", err)
	}

	// Don't call main() directly as it calls os.Exit()
	// Instead we'll just test a small part of the functionality directly

	// Create a test config file
	config, err := manager.LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	// Verify the config was loaded correctly
	if config.Email != "test@example.com" {
		t.Errorf("Expected email to be test@example.com, got %s", config.Email)
	}

	if config.AcmeServer != "https://acme-staging-v02.api.letsencrypt.org/directory" {
		t.Errorf("Expected ACME server URL to be the staging one, got %s", config.AcmeServer)
	}

	// Test successful
	t.Log("Integration test passed")
}
