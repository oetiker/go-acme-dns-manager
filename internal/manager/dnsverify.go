package manager

import (
	"context"
	"errors" // Added for errors.As
	"fmt"
	"net"
	"strings"
	"time"
)

// DNSResolver defines the interface for DNS resolution
type DNSResolver interface {
	LookupCNAME(ctx context.Context, host string) (string, error)
}

// DefaultDNSResolver uses the system's default resolver
type DefaultDNSResolver struct {
	Resolver *net.Resolver
}

// LookupCNAME implements the DNSResolver interface using the system resolver
func (r *DefaultDNSResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	return r.Resolver.LookupCNAME(ctx, host)
}

// GetBaseDomain extracts the base domain from a wildcard or regular domain
func GetBaseDomain(domain string) string {
	// Remove wildcard prefix if present
	if strings.HasPrefix(domain, "*.") {
		return strings.TrimPrefix(domain, "*.")
	}
	return domain
}

// VerifyCnameRecord checks if the _acme-challenge CNAME record for the domain
// points to the expected target (the fulldomain from acme-dns).
// Exported function
func VerifyCnameRecord(cfg *Config, domain string, expectedTarget string) (bool, error) {
	baseDomain := GetBaseDomain(domain)
	challengeDomain := fmt.Sprintf("_acme-challenge.%s", baseDomain)
	expectedTarget = strings.TrimSuffix(expectedTarget, ".") // Ensure no trailing dot for comparison

	DefaultLogger.Infof("Verifying CNAME record for %s -> %s", challengeDomain, expectedTarget)

	var resolver DNSResolver

	if cfg.DnsResolver != "" {
		DefaultLogger.Infof("Using custom DNS resolver: %s", cfg.DnsResolver)
		// Ensure the resolver address includes a port
		resolverAddr := cfg.DnsResolver
		if !strings.Contains(resolverAddr, ":") {
			resolverAddr += ":53" // Default DNS port
		}

		customResolver := &net.Resolver{
			PreferGo: true, // Use Go's built-in resolver
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Second * 10, // Timeout for dialing the resolver
				}
				// Ignore the address passed in, use the configured one
				return d.DialContext(ctx, "udp", resolverAddr)
			},
		}
		resolver = &DefaultDNSResolver{Resolver: customResolver}
	} else {
		DefaultLogger.Infof("Using system default DNS resolver")
		// Use default resolver
		resolver = &DefaultDNSResolver{Resolver: net.DefaultResolver}
	}

	return VerifyWithResolver(resolver, challengeDomain, expectedTarget)
}

// VerifyWithResolver performs the actual CNAME verification with the provided resolver
// This function allows for easier testing with mock resolvers
// Exported for testing
func VerifyWithResolver(resolver DNSResolver, challengeDomain string, expectedTarget string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultDNSTimeout*time.Second) // Overall timeout for lookup
	defer cancel()

	cname, err := resolver.LookupCNAME(ctx, challengeDomain)
	if err != nil {
		// Check for specific error types, like "no such host" which means the record doesn't exist
		var dnsErr *net.DNSError
		if ok := errors.As(err, &dnsErr); ok && dnsErr.IsNotFound {
			DefaultLogger.Warnf("CNAME record for %s not found.", challengeDomain)
			return false, nil // Record not found is a valid check result (false), not an error
		}
		// Other errors (timeout, server failure) are actual errors
		DefaultLogger.Errorf("Error looking up CNAME for %s: %v", challengeDomain, err)
		return false, fmt.Errorf("DNS lookup error for %s: %w", challengeDomain, err)
	}

	cname = strings.TrimSuffix(cname, ".") // Ensure no trailing dot
	DefaultLogger.Infof("Found CNAME for %s: %s", challengeDomain, cname)

	isValid := cname == expectedTarget
	if isValid {
		DefaultLogger.Infof("CNAME record for %s is valid.", challengeDomain)
	} else {
		DefaultLogger.Warnf("CNAME record for %s is INVALID (Expected: %s, Found: %s)", challengeDomain, expectedTarget, cname)
	}

	return isValid, nil
}
