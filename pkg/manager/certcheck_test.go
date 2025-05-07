package manager

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCertificateNeedsRenewal(t *testing.T) {
	// Create a temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "certcheck-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: failed to remove temp directory: %v", err)
		}
	}()

	// Create test certificates with various expiration dates
	// This is a simplified version that just sets up the test files
	// In a real test, we'd use cryptographic functions to create proper certificates

	// Test certificate paths
	expiredCertPath := filepath.Join(tempDir, "expired.crt")
	nearExpirationCertPath := filepath.Join(tempDir, "near_expiration.crt")
	validCertPath := filepath.Join(tempDir, "valid.crt")
	domainMismatchCertPath := filepath.Join(tempDir, "domain_mismatch.crt")

	// Just create the test files for now - we'll let the CertificateNeedsRenewal function
	// handle the loading/parsing errors
	testFiles := []string{expiredCertPath, nearExpirationCertPath, validCertPath, domainMismatchCertPath}
	for _, file := range testFiles {
		if f, err := os.Create(file); err == nil {
			if err := f.Close(); err != nil {
				t.Fatalf("Failed to close file %s: %v", file, err)
			}
		} else {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Define test cases
	tests := []struct {
		name              string
		certPath          string
		requestedDomains  []string
		renewalThreshold  time.Duration
		expectRenewal     bool
		expectReasonMatch string
	}{
		{
			name:              "Missing Certificate",
			certPath:          filepath.Join(tempDir, "nonexistent.crt"),
			requestedDomains:  []string{"example.com"},
			renewalThreshold:  30 * 24 * time.Hour,
			expectRenewal:     true,
			expectReasonMatch: "could not read certificate file",
		},
		{
			name:              "Invalid Certificate Format",
			certPath:          expiredCertPath, // We'll use one of our empty files
			requestedDomains:  []string{"example.com"},
			renewalThreshold:  30 * 24 * time.Hour,
			expectRenewal:     true,
			expectReasonMatch: "failed to decode PEM block",
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsRenewal, reason, err := CertificateNeedsRenewal(
				tt.certPath, tt.requestedDomains, tt.renewalThreshold)

			if needsRenewal != tt.expectRenewal {
				t.Errorf("CertificateNeedsRenewal() needsRenewal = %v, want %v",
					needsRenewal, tt.expectRenewal)
			}

			if tt.expectReasonMatch != "" && reason == "" {
				t.Errorf("CertificateNeedsRenewal() reason is empty, want match for %q",
					tt.expectReasonMatch)
			}

			// For certain test cases, we expect an error
			if tt.name == "Missing Certificate" && err == nil {
				t.Errorf("CertificateNeedsRenewal() expected error for missing certificate, got nil")
			}
		})
	}
}

func TestCompareCertificateDomains(t *testing.T) {
	// Create a test certificate with specific domains
	testCert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "example.com",
		},
		DNSNames: []string{
			"example.com",
			"www.example.com",
			"api.example.com",
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),
	}

	tests := []struct {
		name           string
		requestDomains []string
		wantMissing    []string
		wantExtra      []string
	}{
		{
			name:           "Exact Match",
			requestDomains: []string{"example.com", "www.example.com", "api.example.com"},
			wantMissing:    nil,
			wantExtra:      nil,
		},
		{
			name:           "Missing Domains",
			requestDomains: []string{"example.com", "www.example.com", "api.example.com", "new.example.com"},
			wantMissing:    []string{"new.example.com"},
			wantExtra:      nil,
		},
		{
			name:           "Extra Domains",
			requestDomains: []string{"example.com", "www.example.com"},
			wantMissing:    nil,
			wantExtra:      []string{"api.example.com"},
		},
		{
			name:           "Both Missing and Extra",
			requestDomains: []string{"example.com", "new.example.com"},
			wantMissing:    []string{"new.example.com"},
			wantExtra:      []string{"www.example.com", "api.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			missing, extra := CompareCertificateDomains(testCert, tt.requestDomains)

			// Check missing domains
			if len(missing) != len(tt.wantMissing) {
				t.Errorf("CompareCertificateDomains() missing domains count = %v, want %v",
					len(missing), len(tt.wantMissing))
			}

			// Check extra domains
			if len(extra) != len(tt.wantExtra) {
				t.Errorf("CompareCertificateDomains() extra domains count = %v, want %v",
					len(extra), len(tt.wantExtra))
			}

			// Check each missing domain
			for _, d := range tt.wantMissing {
				found := false
				for _, md := range missing {
					if md == d {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("CompareCertificateDomains() missing domain %q not found in result", d)
				}
			}

			// Check each extra domain
			for _, d := range tt.wantExtra {
				found := false
				for _, ed := range extra {
					if ed == d {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("CompareCertificateDomains() extra domain %q not found in result", d)
				}
			}
		})
	}
}
