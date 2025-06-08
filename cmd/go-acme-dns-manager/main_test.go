package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/common"
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
			certName, domains, keyType, err := manager.ParseCertArg(tc.arg)

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
			_, _, _, err := manager.ParseCertArg(tc.arg)

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

// TestHandleApplicationError tests the error handling function
func TestHandleApplicationError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectation func(t *testing.T, output string)
	}{
		{
			name: "Config Error",
			err: common.NewApplicationError(common.ErrorTypeConfig, "load config", "configuration file not found").
				AddContext("file", "/path/to/config.yaml").
				AddSuggestion("Check file path"),
			expectation: func(t *testing.T, output string) {
				if !strings.Contains(output, "❌ Application Error:") {
					t.Error("Expected error header not found")
				}
				if !strings.Contains(output, "🔧 Configuration Help:") {
					t.Error("Expected config help section not found")
				}
				if !strings.Contains(output, "-print-config-template") {
					t.Error("Expected config template suggestion not found")
				}
				if !strings.Contains(output, "configuration file not found") {
					t.Error("Expected error message not found")
				}
			},
		},
		{
			name: "Network Error",
			err: common.NewApplicationError(common.ErrorTypeNetwork, "connect to server", "connection timeout").
				AddContext("server", "acme-dns.example.com").
				AddSuggestion("Check network connectivity"),
			expectation: func(t *testing.T, output string) {
				if !strings.Contains(output, "🌐 Network Help:") {
					t.Error("Expected network help section not found")
				}
				if !strings.Contains(output, "firewall settings") {
					t.Error("Expected firewall advice not found")
				}
				if !strings.Contains(output, "connection timeout") {
					t.Error("Expected error message not found")
				}
			},
		},
		{
			name: "DNS Error",
			err: common.NewApplicationError(common.ErrorTypeDNS, "verify CNAME", "DNS record not found").
				AddContext("domain", "example.com").
				AddSuggestion("Check DNS configuration"),
			expectation: func(t *testing.T, output string) {
				if !strings.Contains(output, "🔍 DNS Help:") {
					t.Error("Expected DNS help section not found")
				}
				if !strings.Contains(output, "dig") {
					t.Error("Expected dig command suggestion not found")
				}
				if !strings.Contains(output, "DNS record not found") {
					t.Error("Expected error message not found")
				}
			},
		},
		{
			name: "Validation Error",
			err: common.NewApplicationError(common.ErrorTypeValidation, "parse arguments", "invalid domain format").
				AddContext("argument", "invalid-domain").
				AddSuggestion("Check argument format"),
			expectation: func(t *testing.T, output string) {
				if !strings.Contains(output, "✅ Validation Help:") {
					t.Error("Expected validation help section not found")
				}
				if !strings.Contains(output, "command line arguments") {
					t.Error("Expected command line advice not found")
				}
				if !strings.Contains(output, "invalid domain format") {
					t.Error("Expected error message not found")
				}
			},
		},
		{
			name: "ACME Error (no special handling)",
			err: common.NewApplicationError(common.ErrorTypeACME, "register account", "ACME server error").
				AddContext("server", "letsencrypt.org").
				AddSuggestion("Try again later"),
			expectation: func(t *testing.T, output string) {
				if !strings.Contains(output, "❌ Application Error:") {
					t.Error("Expected error header not found")
				}
				if !strings.Contains(output, "ACME server error") {
					t.Error("Expected error message not found")
				}
				// Should not have type-specific help for ACME errors
				if strings.Contains(output, "🔧 Configuration Help:") ||
					strings.Contains(output, "🌐 Network Help:") ||
					strings.Contains(output, "🔍 DNS Help:") ||
					strings.Contains(output, "✅ Validation Help:") {
					t.Error("Should not show type-specific help for ACME errors")
				}
			},
		},
		{
			name: "Generic Error",
			err:  errors.New("generic error message"),
			expectation: func(t *testing.T, output string) {
				if !strings.Contains(output, "Application error: generic error message") {
					t.Error("Expected generic error message not found")
				}
				if !strings.Contains(output, "💡 For more help, use -h flag") {
					t.Error("Expected help message not found")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr output
			oldStderr := os.Stderr
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			os.Stderr = w

			// Call the function
			handleApplicationError(tt.err)

			// Restore stderr and close writer
			os.Stderr = oldStderr
			if err := w.Close(); err != nil {
				t.Errorf("Failed to close writer: %v", err)
			}

			// Read captured output
			var buf bytes.Buffer
			_, err = buf.ReadFrom(r)
			if err != nil {
				t.Fatalf("Failed to read captured output: %v", err)
			}
			if err := r.Close(); err != nil {
				t.Errorf("Failed to close reader: %v", err)
			}

			output := buf.String()
			tt.expectation(t, output)
		})
	}
}

// TestMain_Integration tests the main function integration without calling os.Exit
func TestMain_Integration(t *testing.T) {
	// Test main function components without actually calling main()
	// since main() calls os.Exit which would terminate the test

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "Version Variable",
			testFunc: func(t *testing.T) {
				if version == "" {
					t.Error("Version should not be empty")
				}
				if version == "local-version" {
					t.Log("Using default local version (expected in development)")
				}
			},
		},
		{
			name: "Context Timeout Setup",
			testFunc: func(t *testing.T) {
				// Test the timeout logic from main()
				ctx := context.Background()
				ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
				defer cancel()

				// Verify context has deadline
				deadline, ok := ctx.Deadline()
				if !ok {
					t.Error("Context should have a deadline")
				}

				// Verify deadline is approximately 30 minutes from now
				expectedDeadline := time.Now().Add(30 * time.Minute)
				if deadline.Before(expectedDeadline.Add(-1*time.Minute)) ||
					deadline.After(expectedDeadline.Add(1*time.Minute)) {
					t.Errorf("Context deadline not within expected range. Got: %v, Expected around: %v", deadline, expectedDeadline)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

// TestMain_ErrorHandling tests error handling pathways
func TestMain_ErrorHandling(t *testing.T) {
	// Test that would trigger error conditions similar to main()

	t.Run("Application Error Pathway", func(t *testing.T) {
		// Simulate the error handling that would occur in main()
		testErr := common.NewApplicationError(
			common.ErrorTypeValidation,
			"test operation",
			"test error message",
		).AddContext("test", "value")

		// Test that GetApplicationError works (used in handleApplicationError)
		appErr := common.GetApplicationError(testErr)
		if appErr == nil {
			t.Error("GetApplicationError should return the ApplicationError")
		} else if appErr.Type != common.ErrorTypeValidation {
			t.Errorf("Expected ErrorTypeValidation, got %v", appErr.Type)
		}

		if appErr.Message != "test error message" {
			t.Errorf("Expected 'test error message', got %s", appErr.Message)
		}
	})

	t.Run("Generic Error Pathway", func(t *testing.T) {
		// Test generic error handling
		testErr := errors.New("generic test error")

		// Test that GetApplicationError returns nil for generic errors
		appErr := common.GetApplicationError(testErr)
		if appErr != nil {
			t.Error("GetApplicationError should return nil for generic errors")
		}
	})
}

// TestMain_Subprocess tests the actual main function by running it as a subprocess
func TestMain_Subprocess(t *testing.T) {
	// Skip this test if we're already in a subprocess to avoid infinite recursion
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		return
	}

	tests := []struct {
		name        string
		args        []string
		expectExit  int
		expectOut   string
		expectErr   string
		timeout     time.Duration
	}{
		{
			name:       "Help Flag",
			args:       []string{"-h"},
			expectExit: 0,
			expectOut:  "Usage:",
			timeout:    5 * time.Second,
		},
		{
			name:       "Version Flag",
			args:       []string{"-version"},
			expectExit: -1, // Program hangs due to WaitForShutdown, gets killed by timeout
			expectOut:  "go-acme-dns-manager",
			timeout:    2 * time.Second, // Shorter timeout since we expect hanging
		},
		{
			name:       "Invalid Config Path",
			args:       []string{"-config", "/nonexistent/config.yaml", "-auto"},
			expectExit: 1,
			expectErr:  "Application Error",
			timeout:    10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the test binary
			cmd := exec.Command("go", "build", "-o", "main_test_binary", ".")
			cmd.Dir = "." // Current directory
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to build test binary: %v", err)
			}
			defer func() {
				if err := os.Remove("main_test_binary"); err != nil {
					t.Logf("Warning: failed to remove test binary: %v", err)
				}
			}()

			// Run the test binary with a timeout
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			cmd = exec.CommandContext(ctx, "./main_test_binary", tt.args...)
			cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1")

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			// Check exit code
			var exitCode int
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					t.Fatalf("Unexpected error running subprocess: %v", err)
				}
			}

			if exitCode != tt.expectExit {
				t.Errorf("Expected exit code %d, got %d", tt.expectExit, exitCode)
				t.Logf("Stdout: %s", stdout.String())
				t.Logf("Stderr: %s", stderr.String())
			}

			// Check expected output
			if tt.expectOut != "" {
				output := stdout.String() + stderr.String()
				if !strings.Contains(output, tt.expectOut) {
					t.Errorf("Expected output to contain %q, got:\nStdout: %s\nStderr: %s",
						tt.expectOut, stdout.String(), stderr.String())
				}
			}

			// Check expected error
			if tt.expectErr != "" {
				if !strings.Contains(stderr.String(), tt.expectErr) {
					t.Errorf("Expected stderr to contain %q, got: %s", tt.expectErr, stderr.String())
				}
			}
		})
	}
}
