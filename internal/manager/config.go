package manager

import (
	"encoding/json"
	"fmt"
	"io" // Added for io.Writer
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

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

// AutoDomainsConfig holds the configuration for automatic renewal.
type AutoDomainsConfig struct {
	GraceDays int                   `yaml:"grace_days"` // Renewal window in days
	Certs     map[string]CertConfig `yaml:"certs"`      // Map: cert-name -> {domains: [...], key_type: "..."}
}

// Config holds the application configuration, loaded from config.yaml.
type Config struct {
	Email           string             `yaml:"email"`
	AcmeServer      string             `yaml:"acme_server"`
	AcmeDnsServer   string             `yaml:"acme_dns_server"`
	DnsResolver     string             `yaml:"dns_resolver,omitempty"`
	CertStoragePath string             `yaml:"cert_storage_path"`
	AutoDomains     *AutoDomainsConfig `yaml:"auto_domains,omitempty"`

	// Internal fields
	configPath string `yaml:"-"`
}

// LoadConfig reads the YAML configuration file from the given path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	// Set default values before unmarshalling
	cfg := &Config{
		configPath:      path,
		CertStoragePath: ".lego", // Default value if not in yaml (keeping '.lego' for now for compatibility?)
		// Consider changing default to ".certs" or similar? For now, keep .lego
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	// Resolve CertStoragePath relative to the config file directory
	configDir := filepath.Dir(path)
	if !filepath.IsAbs(cfg.CertStoragePath) {
		cfg.CertStoragePath = filepath.Join(configDir, cfg.CertStoragePath)
	}

	// Basic validation
	if cfg.Email == "" || cfg.Email == "your-email@example.com" {
		return nil, fmt.Errorf("config error: 'email' must be set and not placeholder")
	}
	if cfg.AcmeServer == "" {
		return nil, fmt.Errorf("config error: 'acme_server' must be set")
	}
	if cfg.AcmeDnsServer == "" {
		return nil, fmt.Errorf("config error: 'acme_dns_server' must be set")
	}
	if cfg.CertStoragePath == "" {
		return nil, fmt.Errorf("config error: 'cert_storage_path' must be set")
	}

	// Validation for auto_domains section if present
	if cfg.AutoDomains != nil {
		if cfg.AutoDomains.GraceDays <= 0 {
			cfg.AutoDomains.GraceDays = DefaultGraceDays
			log.Printf("Warning: auto_domains.grace_days not set or invalid in config, defaulting to %d days.", DefaultGraceDays)
		}
		if len(cfg.AutoDomains.Certs) == 0 {
			log.Printf("Warning: auto_domains section found in config, but 'certs' map is empty or missing.")
		}
		for name, cert := range cfg.AutoDomains.Certs {
			if len(cert.Domains) == 0 {
				return nil, fmt.Errorf("config error: auto_domains.certs['%s'] must have at least one domain in its 'domains' list", name)
			}
			// Validate key_type if specified
			if cert.KeyType != "" && !isValidKeyType(cert.KeyType) {
				return nil, fmt.Errorf("config error: auto_domains.certs['%s'] has invalid key_type: '%s'", name, cert.KeyType)
			}
		}
	}

	return cfg, nil
}

// GenerateDefaultConfig writes a default config template to the provided writer.
func GenerateDefaultConfig(writer io.Writer) error {
	// No need to create directory when writing to stdout/writer

	defaultContent := `# Configuration for go-acme-dns-manager

# Email address for Let's Encrypt registration and notifications
email: "your-email@example.com" # <-- EDIT THIS

# Let's Encrypt ACME server URL
# Production: https://acme-v02.api.letsencrypt.org/directory
# Staging: https://acme-staging-v02.api.letsencrypt.org/directory
acme_server: "https://acme-staging-v02.api.letsencrypt.org/directory" # <-- Use production URL when ready (Renamed from lego_server)

# Key type for the certificate (e.g., rsa2048, rsa4096, ec256, ec384)
key_type: "ec256"

# URL of your acme-dns server (e.g., https://acme-dns.example.com)
acme_dns_server: "https://acme-dns.oetiker.ch" # <-- EDIT THIS if different

# DNS resolver to use for CNAME verification checks (optional, uses system default if empty)
# Example: "1.1.1.1:53" or "8.8.8.8"
dns_resolver: ""

# Path where Let's Encrypt certificates, account info, and acme-dns credentials will be stored.
# Relative paths are relative to the directory containing this config file.
# Default is '.lego' inside the config file directory.
cert_storage_path: ".lego" # <-- Renamed from lego_storage_path

# Storage for acme-dns account credentials is now in a separate JSON file:
# See '<cert_storage_path>/acme-dns-accounts.json'

# Optional section for configuring automatic renewals via the -auto flag.
# If this section is present and -auto is used, the tool will check
# certificates defined here and renew them if they expire within 'graceDays'.
#autoDomains:
#  graceDays: 30 # Renew certs expiring within this many days (default: 30)
#  certs:
#    # The key here (e.g., 'my-main-site') is the name used for certificate files
#    # stored in '<cert_storage_path>/certificates/my-main-site.crt' etc.
#    my-main-site:
#      key_type: "ec256"       # Optional: Override global key_type for this cert
#      domains:
#        - example.com         # First domain is the Common Name (CN)
#        - www.example.com
#    another-service:
#      domains:
#        - service.example.com
`
	_, err := writer.Write([]byte(defaultContent))
	if err != nil {
		return fmt.Errorf("writing default config: %w", err)
	}
	return nil
}

// --- Account Management (Uses separate JSON file) ---

// accountStore holds the accounts and provides thread-safe access.
type accountStore struct {
	filePath string
	accounts map[string]AcmeDnsAccount
	mu       sync.RWMutex
}

// NewAccountStore creates a new store and loads accounts from the file.
func NewAccountStore(filePath string) (*accountStore, error) {
	store := &accountStore{
		filePath: filePath,
		accounts: make(map[string]AcmeDnsAccount),
	}
	err := store.loadAccounts()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return store, nil
}

// loadAccounts reads the JSON account file. Not exported.
func (s *accountStore) loadAccounts() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.accounts = make(map[string]AcmeDnsAccount)
			return nil
		}
		return fmt.Errorf("reading accounts file %s: %w", s.filePath, err)
	}

	if len(data) == 0 {
		s.accounts = make(map[string]AcmeDnsAccount)
		return nil
	}

	err = json.Unmarshal(data, &s.accounts)
	if err != nil {
		return fmt.Errorf("parsing accounts file %s: %w", s.filePath, err)
	}

	if s.accounts == nil {
		s.accounts = make(map[string]AcmeDnsAccount)
	}

	return nil
}

// SaveAccounts writes the current accounts map back to the JSON file. Exported method.
func (s *accountStore) SaveAccounts() error {
	s.mu.RLock()
	accountsCopy := make(map[string]AcmeDnsAccount, len(s.accounts))
	for k, v := range s.accounts {
		accountsCopy[k] = v
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(accountsCopy, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling accounts: %w", err)
	}

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, DirPermissions); err != nil {
		return fmt.Errorf("creating directory %s for accounts file: %w", dir, err)
	}

	err = os.WriteFile(s.filePath, data, PrivateKeyPermissions)
	if err != nil {
		return fmt.Errorf("writing accounts file %s: %w", s.filePath, err)
	}
	return nil
}

// GetAccount retrieves an account thread-safely. Exported method.
func (s *accountStore) GetAccount(domain string) (AcmeDnsAccount, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.accounts == nil {
		return AcmeDnsAccount{}, false
	}
	acc, ok := s.accounts[domain]
	return acc, ok
}

// SetAccount sets an account thread-safely. Exported method.
func (s *accountStore) SetAccount(domain string, account AcmeDnsAccount) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.accounts == nil {
		s.accounts = make(map[string]AcmeDnsAccount)
	}
	s.accounts[domain] = account
}

// GetAllAccounts returns a copy of all accounts. Exported method.
func (s *accountStore) GetAllAccounts() map[string]AcmeDnsAccount {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.accounts == nil {
		return make(map[string]AcmeDnsAccount)
	}
	accountsCopy := make(map[string]AcmeDnsAccount, len(s.accounts))
	for k, v := range s.accounts {
		accountsCopy[k] = v
	}
	return accountsCopy
}

// Helper function to get the renewal threshold duration
func (cfg *Config) GetRenewalThreshold() time.Duration {
	days := DefaultGraceDays
	if cfg.AutoDomains != nil && cfg.AutoDomains.GraceDays > 0 {
		days = cfg.AutoDomains.GraceDays
	}
	return time.Duration(days) * 24 * time.Hour
}

// isValidKeyType checks if a key type is valid for certificate usage
func isValidKeyType(keyType string) bool {
	validTypes := []string{"rsa2048", "rsa3072", "rsa4096", "ec256", "ec384"}
	for _, valid := range validTypes {
		if keyType == valid {
			return true
		}
	}
	return false
}
