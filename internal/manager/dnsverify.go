package manager // Changed from main

import (
	"context"
	"errors" // Added for errors.As
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// VerifyCnameRecord checks if the _acme-challenge CNAME record for the domain
// points to the expected target (the fulldomain from acme-dns).
// Exported function
func VerifyCnameRecord(cfg *Config, domain string, expectedTarget string) (bool, error) {
	challengeDomain := fmt.Sprintf("_acme-challenge.%s", domain)
	expectedTarget = strings.TrimSuffix(expectedTarget, ".") // Ensure no trailing dot for comparison

	log.Printf("Verifying CNAME record for %s -> %s", challengeDomain, expectedTarget)

	var resolver *net.Resolver
	if cfg.DnsResolver != "" {
		log.Printf("Using custom DNS resolver: %s", cfg.DnsResolver)
		// Ensure the resolver address includes a port
		resolverAddr := cfg.DnsResolver
		if !strings.Contains(resolverAddr, ":") {
			resolverAddr += ":53" // Default DNS port
		}
		resolver = &net.Resolver{
			PreferGo: true, // Use Go's built-in resolver
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Second * 10, // Timeout for dialing the resolver
				}
				// Ignore the address passed in, use the configured one
				return d.DialContext(ctx, "udp", resolverAddr)
			},
		}
	} else {
		log.Printf("Using system default DNS resolver")
		// Use default resolver
		resolver = net.DefaultResolver
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // Overall timeout for lookup
	defer cancel()

	cname, err := resolver.LookupCNAME(ctx, challengeDomain)
	if err != nil {
		// Check for specific error types, like "no such host" which means the record doesn't exist
		var dnsErr *net.DNSError
		if ok := errors.As(err, &dnsErr); ok && dnsErr.IsNotFound {
			log.Printf("CNAME record for %s not found.", challengeDomain)
			return false, nil // Record not found is a valid check result (false), not an error
		}
		// Other errors (timeout, server failure) are actual errors
		log.Printf("Error looking up CNAME for %s: %v", challengeDomain, err)
		return false, fmt.Errorf("DNS lookup error for %s: %w", challengeDomain, err)
	}

	cname = strings.TrimSuffix(cname, ".") // Ensure no trailing dot
	log.Printf("Found CNAME for %s: %s", challengeDomain, cname)

	isValid := cname == expectedTarget
	if isValid {
		log.Printf("CNAME record for %s is valid.", challengeDomain)
	} else {
		log.Printf("CNAME record for %s is INVALID (Expected: %s, Found: %s)", challengeDomain, expectedTarget, cname)
	}

	return isValid, nil
}
