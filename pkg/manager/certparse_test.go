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

func TestParseCertRequest(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    CertRequest
		wantErr bool
	}{
		{
			name: "Valid Request",
			arg:  "mycert@example.com,www.example.com/key_type=ec384",
			want: CertRequest{
				Name:    "mycert",
				Domains: []string{"example.com", "www.example.com"},
				KeyType: "ec384",
			},
			wantErr: false,
		},
		{
			name:    "Invalid Request",
			arg:     "my/cert@example.com",
			want:    CertRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCertRequest(tt.arg)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCertRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if got.Name != tt.want.Name {
				t.Errorf("ParseCertRequest() name = %v, want %v", got.Name, tt.want.Name)
			}

			if len(got.Domains) != len(tt.want.Domains) {
				t.Errorf("ParseCertRequest() domains count = %v, want %v", len(got.Domains), len(tt.want.Domains))
				return
			}
			for i, d := range tt.want.Domains {
				if got.Domains[i] != d {
					t.Errorf("ParseCertRequest() domain at index %d = %v, want %v", i, got.Domains[i], d)
				}
			}

			if got.KeyType != tt.want.KeyType {
				t.Errorf("ParseCertRequest() keyType = %v, want %v", got.KeyType, tt.want.KeyType)
			}
		})
	}
}
