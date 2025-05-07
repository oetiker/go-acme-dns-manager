// Package manager provides functions for managing ACME DNS certificates.
package manager

import (
	"fmt"
	"strings"
)

// RequiredCNAME holds information about a required CNAME record
type RequiredCNAME struct {
	Domain      string
	CNAMERecord string
	Target      string
}

// GroupCNAMEsByTarget groups CNAME records by their record name and target for efficient display
// Returns a map structure: Map[CNAMERecord]Map[Target][]Domains
func GroupCNAMEsByTarget(records []RequiredCNAME) map[string]map[string][]string {
	// Create maps to group by CNAME record AND by target
	cnameMap := make(map[string]map[string][]string) // Map[CNAMERecord]Map[Target][]Domains

	// Organize all records
	for _, cname := range records {
		// Initialize the map for this CNAME record if it doesn't exist
		if _, exists := cnameMap[cname.CNAMERecord]; !exists {
			cnameMap[cname.CNAMERecord] = make(map[string][]string)
		}

		// Add this domain to the appropriate target group for this CNAME record
		cnameMap[cname.CNAMERecord][cname.Target] = append(
			cnameMap[cname.CNAMERecord][cname.Target],
			cname.Domain,
		)
	}

	return cnameMap
}

// FormatCNAMERecords formats CNAME records for display in BIND format
func FormatCNAMERecords(cnameGroups map[string]map[string][]string) string {
	var result strings.Builder

	result.WriteString("Add the following CNAME records to your DNS:\n\n")

	// Process each unique CNAME record with its domains grouped by target
	for cnameRecord, targetGroups := range cnameGroups {
		for target, domains := range targetGroups {
			// Create a proper comment showing all domains using this record
			var commentParts []string
			for _, domain := range domains {
				if strings.HasPrefix(domain, "*.") {
					commentParts = append(commentParts, domain+" (wildcard)")
				} else {
					commentParts = append(commentParts, domain)
				}
			}
			comment := strings.Join(commentParts, ", ")

			// Print in BIND format with comment
			result.WriteString(fmt.Sprintf("; %s\n", comment))
			result.WriteString(fmt.Sprintf("%s. IN CNAME %s.\n\n", cnameRecord, target))
		}
	}

	return result.String()
}

// CreateRequiredCNAME creates a RequiredCNAME structure for a domain/target pair
func CreateRequiredCNAME(domain string, target string) RequiredCNAME {
	baseDomain := GetBaseDomain(domain)
	cnameRecord := fmt.Sprintf("_acme-challenge.%s", baseDomain)

	return RequiredCNAME{
		Domain:      domain,
		CNAMERecord: cnameRecord,
		Target:      target,
	}
}
