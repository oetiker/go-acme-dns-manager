package manager

import (
	"testing"
)

func TestGroupCNAMEsByTarget(t *testing.T) {
	tests := []struct {
		name    string
		records []RequiredCNAME
		want    map[string]map[string][]string
	}{
		{
			name:    "Empty Records",
			records: []RequiredCNAME{},
			want:    map[string]map[string][]string{},
		},
		{
			name: "Single Record",
			records: []RequiredCNAME{
				{
					Domain:      "example.com",
					CNAMERecord: "_acme-challenge.example.com",
					Target:      "abcdef.acme-dns.com",
				},
			},
			want: map[string]map[string][]string{
				"_acme-challenge.example.com": {
					"abcdef.acme-dns.com": []string{"example.com"},
				},
			},
		},
		{
			name: "Multiple Records Same CNAME Different Targets",
			records: []RequiredCNAME{
				{
					Domain:      "example.com",
					CNAMERecord: "_acme-challenge.example.com",
					Target:      "abcdef.acme-dns.com",
				},
				{
					Domain:      "example.org",
					CNAMERecord: "_acme-challenge.example.org",
					Target:      "ghijkl.acme-dns.com",
				},
			},
			want: map[string]map[string][]string{
				"_acme-challenge.example.com": {
					"abcdef.acme-dns.com": []string{"example.com"},
				},
				"_acme-challenge.example.org": {
					"ghijkl.acme-dns.com": []string{"example.org"},
				},
			},
		},
		{
			name: "Multiple Records Same CNAME Same Target",
			records: []RequiredCNAME{
				{
					Domain:      "example.com",
					CNAMERecord: "_acme-challenge.example.com",
					Target:      "abcdef.acme-dns.com",
				},
				{
					Domain:      "www.example.com",
					CNAMERecord: "_acme-challenge.example.com",
					Target:      "abcdef.acme-dns.com",
				},
			},
			want: map[string]map[string][]string{
				"_acme-challenge.example.com": {
					"abcdef.acme-dns.com": []string{"example.com", "www.example.com"},
				},
			},
		},
		{
			name: "Multiple Records Same CNAME Different Targets",
			records: []RequiredCNAME{
				{
					Domain:      "example.com",
					CNAMERecord: "_acme-challenge.example.com",
					Target:      "abcdef.acme-dns.com",
				},
				{
					Domain:      "www.example.com",
					CNAMERecord: "_acme-challenge.example.com",
					Target:      "ghijkl.acme-dns.com",
				},
			},
			want: map[string]map[string][]string{
				"_acme-challenge.example.com": {
					"abcdef.acme-dns.com": []string{"example.com"},
					"ghijkl.acme-dns.com": []string{"www.example.com"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupCNAMEsByTarget(tt.records)

			// Check overall structure matches (counts)
			if len(got) != len(tt.want) {
				t.Errorf("GroupCNAMEsByTarget() returned %d CNAME records, want %d",
					len(got), len(tt.want))
			}

			// Check each CNAME record
			for cnameRecord, wantTargetGroups := range tt.want {
				gotTargetGroups, exists := got[cnameRecord]
				if !exists {
					t.Errorf("GroupCNAMEsByTarget() missing CNAME record %q", cnameRecord)
					continue
				}

				if len(gotTargetGroups) != len(wantTargetGroups) {
					t.Errorf("GroupCNAMEsByTarget() returned %d target groups for CNAME record %q, want %d",
						len(gotTargetGroups), cnameRecord, len(wantTargetGroups))
				}

				// Check each target group
				for target, wantDomains := range wantTargetGroups {
					gotDomains, targetExists := gotTargetGroups[target]
					if !targetExists {
						t.Errorf("GroupCNAMEsByTarget() missing target %q for CNAME record %q",
							target, cnameRecord)
						continue
					}

					if len(gotDomains) != len(wantDomains) {
						t.Errorf("GroupCNAMEsByTarget() returned %d domains for CNAME record %q target %q, want %d",
							len(gotDomains), cnameRecord, target, len(wantDomains))
					}

					// Check each domain in the target group
					for _, domain := range wantDomains {
						found := false
						for _, gotDomain := range gotDomains {
							if gotDomain == domain {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("GroupCNAMEsByTarget() missing domain %q for CNAME record %q target %q",
								domain, cnameRecord, target)
						}
					}
				}
			}
		})
	}
}

func TestCreateRequiredCNAME(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		target string
		want   RequiredCNAME
	}{
		{
			name:   "Simple Domain",
			domain: "example.com",
			target: "abcdef.acme-dns.com",
			want: RequiredCNAME{
				Domain:      "example.com",
				CNAMERecord: "_acme-challenge.example.com",
				Target:      "abcdef.acme-dns.com",
			},
		},
		{
			name:   "Subdomain",
			domain: "sub.example.com",
			target: "ghijkl.acme-dns.com",
			want: RequiredCNAME{
				Domain:      "sub.example.com",
				CNAMERecord: "_acme-challenge.sub.example.com",
				Target:      "ghijkl.acme-dns.com",
			},
		},
		{
			name:   "Wildcard Domain",
			domain: "*.example.com",
			target: "mnopqr.acme-dns.com",
			want: RequiredCNAME{
				Domain:      "*.example.com",
				CNAMERecord: "_acme-challenge.example.com", // Base domain for wildcard
				Target:      "mnopqr.acme-dns.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateRequiredCNAME(tt.domain, tt.target)

			if got.Domain != tt.want.Domain {
				t.Errorf("CreateRequiredCNAME() Domain = %q, want %q", got.Domain, tt.want.Domain)
			}
			if got.CNAMERecord != tt.want.CNAMERecord {
				t.Errorf("CreateRequiredCNAME() CNAMERecord = %q, want %q", got.CNAMERecord, tt.want.CNAMERecord)
			}
			if got.Target != tt.want.Target {
				t.Errorf("CreateRequiredCNAME() Target = %q, want %q", got.Target, tt.want.Target)
			}
		})
	}
}
