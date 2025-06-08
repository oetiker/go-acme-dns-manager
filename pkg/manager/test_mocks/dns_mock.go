//go:build testutils
// +build testutils

package test_mocks

import (
	"context"
	"net"
)

// MockDNSResolver is a DNS resolver that returns predefined responses
type MockDNSResolver struct {
	// Map of hostname to CNAME response
	CNAMEResponses map[string]string
	// Map of hostname to error response
	ErrorResponses map[string]error
}

// LookupCNAME implements the DNSResolver interface
func (r *MockDNSResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	// Check if we have a predefined error for this host
	if err, ok := r.ErrorResponses[host]; ok {
		return "", err
	}

	// Check if we have a predefined CNAME for this host
	if cname, ok := r.CNAMEResponses[host]; ok {
		return cname, nil
	}

	// Default to "no such host" error
	return "", &net.DNSError{
		Err:        "no such host",
		Name:       host,
		IsNotFound: true,
	}
}

// NewMockDNSResolver creates a new mock DNS resolver
func NewMockDNSResolver() *MockDNSResolver {
	return &MockDNSResolver{
		CNAMEResponses: make(map[string]string),
		ErrorResponses: make(map[string]error),
	}
}

// AddCNAMERecord adds a CNAME record to the mock resolver
func (r *MockDNSResolver) AddCNAMERecord(hostname, cname string) {
	r.CNAMEResponses[hostname] = cname
}

// AddErrorRecord adds an error response to the mock resolver
func (r *MockDNSResolver) AddErrorRecord(hostname string, err error) {
	r.ErrorResponses[hostname] = err
}
