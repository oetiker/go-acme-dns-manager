// Package manager provides functions for managing ACME DNS certificates.
package manager

import (
	"fmt"
	"strings"
)

// CertRequest holds parsed certificate request information
type CertRequest struct {
	Name    string
	Domains []string
	KeyType string
}

// ParseCertArg parses certificate arguments in the format cert-name@domain1,domain2/key_type=ec384
// This extracts certificate name, domains list, and optional key type parameter
func ParseCertArg(arg string) (string, []string, string, error) {
	// Check for key_type parameter
	keyType := ""
	domainPart := arg

	// Special case: Check for slash in the cert name part, which is an invalid format
	// Must handle this before processing parameters
	atIndex := strings.Index(arg, "@")
	slashIndex := strings.Index(arg, "/")
	if slashIndex >= 0 && (atIndex == -1 || slashIndex < atIndex) {
		// There's a slash before the @ sign or there's no @ but there is a slash
		// This is only allowed if it's a parameter after the domain part
		return "", nil, "", fmt.Errorf("invalid format: unexpected '/' in certificate name part")
	}

	// Now process any parameters that appear after the domain part
	if strings.Contains(arg, "/") {
		argParts := strings.Split(arg, "/")
		domainPart = argParts[0]

		// Process any parameters after the slash
		for i := 1; i < len(argParts); i++ {
			param := argParts[i]
			if strings.HasPrefix(param, "key_type=") {
				keyType = strings.TrimPrefix(param, "key_type=")
			}
			// No logging in this function - caller should log if needed
		}
	}

	// Simple domain format (no @ symbol) - use as both cert name and domain
	if !strings.Contains(domainPart, "@") {
		// Basic validation for the domain
		if strings.ContainsAny(domainPart, "/\\") {
			return "", nil, "", fmt.Errorf("invalid domain name '%s': must not contain '/' or '\\'", domainPart)
		}
		if domainPart == "" {
			return "", nil, "", fmt.Errorf("empty domain name")
		}
		// Advanced RFC validation for DNS names
		if !IsValidDNSName(domainPart) {
			return "", nil, "", fmt.Errorf("invalid domain name '%s': does not conform to DNS name standards", domainPart)
		}
		return domainPart, []string{domainPart}, keyType, nil
	}

	// Process explicit cert-name@domain format
	parts := strings.SplitN(domainPart, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", nil, "", fmt.Errorf("invalid format: expected 'cert-name@domain1,domain2,...', got '%s'", domainPart)
	}

	certName := parts[0]
	domains := []string{}
	rawDomains := strings.Split(parts[1], ",")
	for _, d := range rawDomains {
		trimmed := strings.TrimSpace(d)
		if trimmed != "" {
			// Validate the domain according to DNS standards
			if !IsValidDNSName(trimmed) {
				return "", nil, "", fmt.Errorf("invalid domain name '%s': does not conform to DNS name standards", trimmed)
			}
			domains = append(domains, trimmed)
		}
	}

	if len(domains) == 0 {
		return "", nil, "", fmt.Errorf("no valid domains found after '@' in argument '%s'", domainPart)
	}

	// Basic validation for cert name
	if strings.ContainsAny(certName, "/\\") {
		return "", nil, "", fmt.Errorf("invalid certificate name '%s': must not contain '/' or '\\'", certName)
	}

	return certName, domains, keyType, nil
}
