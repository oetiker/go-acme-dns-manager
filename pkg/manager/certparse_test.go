package manager

import (
	"testing"
)

func TestParseCertArg(t *testing.T) {
	tests := []struct {
		name        string
		arg         string
		wantName    string
		wantDomains []string
		wantKeyType string
		wantErr     bool
	}{
		{
			name:        "Simple Domain",
			arg:         "example.com",
			wantName:    "example.com",
			wantDomains: []string{"example.com"},
			wantKeyType: "",
			wantErr:     false,
		},
		{
			name:        "Domain With @ Separator",
			arg:         "mycert@example.com",
			wantName:    "mycert",
			wantDomains: []string{"example.com"},
			wantKeyType: "",
			wantErr:     false,
		},
		{
			name:        "Multiple Domains",
			arg:         "mycert@example.com,www.example.com",
			wantName:    "mycert",
			wantDomains: []string{"example.com", "www.example.com"},
			wantKeyType: "",
			wantErr:     false,
		},
		{
			name:        "With Key Type",
			arg:         "mycert@example.com/key_type=ec384",
			wantName:    "mycert",
			wantDomains: []string{"example.com"},
			wantKeyType: "ec384",
			wantErr:     false,
		},
		{
			name:        "Multiple Domains With Key Type",
			arg:         "mycert@example.com,www.example.com/key_type=rsa2048",
			wantName:    "mycert",
			wantDomains: []string{"example.com", "www.example.com"},
			wantKeyType: "rsa2048",
			wantErr:     false,
		},
		{
			name:        "Wildcard Domain",
			arg:         "mycert@*.example.com",
			wantName:    "mycert",
			wantDomains: []string{"*.example.com"},
			wantKeyType: "",
			wantErr:     false,
		},
		{
			name:        "Invalid - Slash In Name",
			arg:         "my/cert@example.com",
			wantName:    "",
			wantDomains: nil,
			wantKeyType: "",
			wantErr:     true,
		},
		{
			name:        "Invalid - Empty Domain After @",
			arg:         "mycert@",
			wantName:    "",
			wantDomains: nil,
			wantKeyType: "",
			wantErr:     true,
		},
		{
			name:        "Invalid - Empty Certificate Name",
			arg:         "@example.com",
			wantName:    "",
			wantDomains: nil,
			wantKeyType: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotDomains, gotKeyType, err := ParseCertArg(tt.arg)

			// Check error condition
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCertArg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Don't check other fields if we expected an error
			}

			// Check certificate name
			if gotName != tt.wantName {
				t.Errorf("ParseCertArg() name = %v, want %v", gotName, tt.wantName)
			}

			// Check domains list
			if len(gotDomains) != len(tt.wantDomains) {
				t.Errorf("ParseCertArg() domains count = %v, want %v", len(gotDomains), len(tt.wantDomains))
				return
			}
			for i, d := range tt.wantDomains {
				if gotDomains[i] != d {
					t.Errorf("ParseCertArg() domain at index %d = %v, want %v", i, gotDomains[i], d)
				}
			}

			// Check key type
			if gotKeyType != tt.wantKeyType {
				t.Errorf("ParseCertArg() keyType = %v, want %v", gotKeyType, tt.wantKeyType)
			}
		})
	}
}
