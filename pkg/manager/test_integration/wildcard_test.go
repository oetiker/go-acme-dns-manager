//go:build testutils
// +build testutils

package test_integration

import (
	"testing"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager/test_mocks"
)

// TestWildcardDomainVerification specifically tests DNS verification with wildcard domains
func TestWildcardDomainVerification(t *testing.T) {
	// Create a mock resolver
	mockResolver := test_mocks.NewMockDNSResolver()

	// Add test records - note that the challenge domain for *.example.com should be _acme-challenge.example.com
	mockResolver.AddCNAMERecord("_acme-challenge.example.com", "valid.acme-dns.example.org")

	// Test the entire verification flow from domain to challenge resolution
	testCases := []struct {
		name           string
		domain         string
		expectedTarget string
		wantValid      bool
	}{
		{
			name:           "Regular domain",
			domain:         "example.com",
			expectedTarget: "valid.acme-dns.example.org",
			wantValid:      true,
		},
		{
			name:           "Wildcard domain",
			domain:         "*.example.com",
			expectedTarget: "valid.acme-dns.example.org",
			wantValid:      true,
		},
		{
			name:           "Multiple domains in same cert",
			domain:         "example.com",
			expectedTarget: "valid.acme-dns.example.org",
			wantValid:      true,
		},
		{
			name:           "Wrong target",
			domain:         "*.example.com",
			expectedTarget: "wrong.acme-dns.example.org",
			wantValid:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the GetBaseDomain function first
			baseDomain := manager.GetBaseDomain(tc.domain)
			if tc.domain == "*.example.com" && baseDomain != "example.com" {
				t.Errorf("GetBaseDomain(%q) = %q, expected %q", tc.domain, baseDomain, "example.com")
				return
			}

			// Bypass the actual DNS resolver by directly testing VerifyWithResolver
			challengeDomain := manager.GetChallengeSubdomain(baseDomain)
			valid, err := manager.VerifyWithResolver(mockResolver, challengeDomain, tc.expectedTarget)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check if the result matches our expectation
			if valid != tc.wantValid {
				t.Errorf("Expected valid=%v, got %v for domain %s", tc.wantValid, valid, tc.domain)
			}
		})
	}
}
