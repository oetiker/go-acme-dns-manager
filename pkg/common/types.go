package common

// AcmeDnsAccount holds the credentials for a specific domain registered with acme-dns.
type AcmeDnsAccount struct {
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	FullDomain string   `json:"fulldomain"`
	SubDomain  string   `json:"subdomain"`
	AllowFrom  []string `json:"allowfrom"`
}

// CertConfig defines a certificate configuration with its associated domains and optional key type.
type CertConfig struct {
	Domains []string `yaml:"domains"`
	KeyType string   `yaml:"key_type,omitempty"` // Optional: Certificate-specific key type
}

// CertRequest represents a certificate request with domains and parameters
type CertRequest struct {
	CertName string
	Domains  []string
	KeyType  string
}
