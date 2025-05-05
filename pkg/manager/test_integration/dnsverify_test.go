package test_integration

import (
	"net"
	"testing"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager/test_mocks"
)

// TestDNSVerification tests the DNS verification functionality with a mock resolver
func TestDNSVerification(t *testing.T) {
	// Create a mock DNS resolver
	mockResolver := test_mocks.NewMockDNSResolver()

	// Add test records
	mockResolver.AddCNAMERecord("_acme-challenge.example.com", "valid.acme-dns.example.org")
	mockResolver.AddCNAMERecord("_acme-challenge.invalid.com", "wrong.acme-dns.example.org")
	// Add wildcard domain record - note that the challenge domain should be _acme-challenge.example.com (no wildcard)
	mockResolver.AddCNAMERecord("_acme-challenge.example.com", "valid.acme-dns.example.org")
	mockResolver.AddErrorRecord("_acme-challenge.error.com", &net.DNSError{
		Err:  "server failure",
		Name: "_acme-challenge.error.com",
	})

	testCases := []struct {
		name            string
		challengeDomain string
		expectedTarget  string
		wantValid       bool
		wantErr         bool
	}{
		{
			name:            "Valid CNAME",
			challengeDomain: "_acme-challenge.example.com",
			expectedTarget:  "valid.acme-dns.example.org",
			wantValid:       true,
			wantErr:         false,
		},
		{
			name:            "Invalid CNAME",
			challengeDomain: "_acme-challenge.invalid.com",
			expectedTarget:  "expected.acme-dns.example.org",
			wantValid:       false,
			wantErr:         false,
		},
		{
			name:            "Missing CNAME",
			challengeDomain: "_acme-challenge.missing.com",
			expectedTarget:  "anything.acme-dns.example.org",
			wantValid:       false,
			wantErr:         false,
		},
		{
			name:            "DNS Error",
			challengeDomain: "_acme-challenge.error.com",
			expectedTarget:  "anything.acme-dns.example.org",
			wantValid:       false,
			wantErr:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid, err := manager.VerifyWithResolver(mockResolver, tc.challengeDomain, tc.expectedTarget)

			// Check error
			if tc.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check validity
			if valid != tc.wantValid {
				t.Errorf("Expected valid=%v, got %v", tc.wantValid, valid)
			}
		})
	}
}
