// filepath: /home/oetiker/checkouts/go-acme-dns-manager/cmd/go-acme-dns-manager/wildcard_cname_test.go
package main

import (
	"strings"
	"testing"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// TestWildcardDomainSkipping tests the logic that prevents duplicate CNAME requirements
// for wildcard domains when the base domain has already been processed.
// This test directly verifies the fix for the bug where the tool would ask users
// to add redundant CNAME records for wildcard domains even when the base domain's
// CNAME was already properly configured.
func TestWildcardDomainSkipping(t *testing.T) {
	// Create a map to track which base domains have been checked
	checkedBaseDomains := make(map[string]bool)

	// Test case 1: Base domain is already checked, should skip wildcard verification
	// -------------------------------------------------------------------

	// Mark base domain as checked
	baseDomain := "example.com"
	checkedBaseDomains[baseDomain] = true

	// Process wildcard domain - should skip verification
	wildcardDomain := "*.example.com"
	wBaseDomain := manager.GetBaseDomain(wildcardDomain)

	// Check if the condition matches what we implemented in main.go
	shouldSkip := strings.HasPrefix(wildcardDomain, "*.") && checkedBaseDomains[wBaseDomain]

	if !shouldSkip {
		t.Errorf("Expected to skip CNAME verification for %s when base domain %s is checked",
			wildcardDomain, baseDomain)
	}

	// Test case 2: Regular (non-wildcard) domain should not be skipped
	// -------------------------------------------------------------------
	regularDomain := "sub.example.com"
	regularBaseDomain := manager.GetBaseDomain(regularDomain)

	shouldSkipRegular := strings.HasPrefix(regularDomain, "*.") && checkedBaseDomains[regularBaseDomain]

	if shouldSkipRegular {
		t.Errorf("Unexpectedly skipping verification for regular domain %s", regularDomain)
	}

	// Test case 3: Wildcard domain with unchecked base should not be skipped
	// -------------------------------------------------------------------
	anotherWildcard := "*.another-example.com"
	anotherBaseDomain := manager.GetBaseDomain(anotherWildcard)

	shouldSkipAnother := strings.HasPrefix(anotherWildcard, "*.") && checkedBaseDomains[anotherBaseDomain]

	if shouldSkipAnother {
		t.Errorf("Unexpectedly skipping verification for %s when base domain %s is not checked",
			anotherWildcard, anotherBaseDomain)
	}

	// Test case 4: Ensure GetBaseDomain correctly extracts the base domain
	// -------------------------------------------------------------------
	extractedBaseDomain := manager.GetBaseDomain("*.test-domain.com")
	if extractedBaseDomain != "test-domain.com" {
		t.Errorf("GetBaseDomain returned %s, expected test-domain.com", extractedBaseDomain)
	}
}
