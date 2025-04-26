package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseCertArg(t *testing.T) {
	testCases := []struct {
		name          string
		arg           string
		wantCertName  string
		wantDomains   []string
		wantErr       bool
		wantErrPrefix string
	}{
		{
			name:         "Valid input with multiple domains",
			arg:          "mycert@example.com,www.example.com",
			wantCertName: "mycert",
			wantDomains:  []string{"example.com", "www.example.com"},
			wantErr:      false,
		},
		{
			name:         "Valid input with single domain",
			arg:          "mycert@example.com",
			wantCertName: "mycert",
			wantDomains:  []string{"example.com"},
			wantErr:      false,
		},
		{
			name:          "Missing @ symbol",
			arg:           "mycert",
			wantCertName:  "",
			wantDomains:   nil,
			wantErr:       true,
			wantErrPrefix: "invalid format:",
		},
		{
			name:          "Empty domains list",
			arg:           "mycert@",
			wantCertName:  "",
			wantDomains:   nil,
			wantErr:       true,
			wantErrPrefix: "invalid format:",
		},
		{
			name:          "Empty cert name",
			arg:           "@example.com",
			wantCertName:  "",
			wantDomains:   nil,
			wantErr:       true,
			wantErrPrefix: "invalid format:",
		},
		{
			name:          "Invalid cert name with slashes",
			arg:           "my/cert@example.com",
			wantCertName:  "",
			wantDomains:   nil,
			wantErr:       true,
			wantErrPrefix: "invalid certificate name",
		},
		{
			name:         "Whitespace trimming in domains",
			arg:          "mycert@example.com, www.example.com ",
			wantCertName: "mycert",
			wantDomains:  []string{"example.com", "www.example.com"},
			wantErr:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			certName, domains, err := parseCertArg(tc.arg)

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
	defer os.Remove(tempFile.Name())

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
	tempFile.Close()

	// Run with --print-config-template which is safe to test
	os.Args = []string{"cmd", "--print-config-template"}

	// Capture stdout to prevent mess in test output
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// This should exit by itself - we're just testing it doesn't crash
	// In a real integration test you'd use a custom main() function that
	// doesn't exit but returns errors instead
	main()

	// Restore stdout (though we don't reach this in the current implementation)
	w.Close()
	os.Stdout = oldStdout
}
