package manager

import (
	"encoding/json"
	"fmt"
	"io" // Added for io.Writer
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kaptinlin/jsonschema"
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

// Config holds the application configuration, loaded from YAML
type Config struct {
	Email            string        `yaml:"email"`
	AcmeServer       string        `yaml:"acme_server"`
	AcmeDnsServer    string        `yaml:"acme_dns_server"`
	DnsResolver      string        `yaml:"dns_resolver,omitempty"`
	CertStoragePath  string        `yaml:"cert_storage_path"`
	ChallengeTimeout time.Duration `yaml:"challenge_timeout,omitempty"` // Timeout for ACME challenges
	HTTPTimeout      time.Duration `yaml:"http_timeout,omitempty"`      // Timeout for HTTP requests to ACME server

	// AutoDomains section for automatic renewals
	AutoDomains *AutoDomainsConfig `yaml:"auto_domains,omitempty"`

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
		configPath:       path,
		CertStoragePath:  ".lego",                 // Default value if not in yaml
		ChallengeTimeout: DefaultChallengeTimeout, // Default challenge timeout
		HTTPTimeout:      DefaultHTTPTimeout,      // Default HTTP timeout
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	// Validate configuration against schema
	if err := validateConfig(data); err != nil {
		return nil, err
	}

	// Resolve CertStoragePath relative to the config file directory
	configDir := filepath.Dir(path)
	if !filepath.IsAbs(cfg.CertStoragePath) {
		cfg.CertStoragePath = filepath.Join(configDir, cfg.CertStoragePath)
	}

	// Check for placeholder email (schema validates that email is present but can't check content)
	if cfg.Email == "your-email@example.com" {
		return nil, fmt.Errorf("config error: 'email' must not be the placeholder value")
	}

	// Additional validation/setup for auto_domains section if present
	if cfg.AutoDomains != nil {
		// Set default grace days if needed (schema ensures it's valid if present)
		if cfg.AutoDomains.GraceDays <= 0 {
			cfg.AutoDomains.GraceDays = DefaultGraceDays
			DefaultLogger.Warnf("Warning: auto_domains.grace_days not set or invalid in config, defaulting to %d days.", DefaultGraceDays)
		}

		// Just provide a warning if certs map is empty
		if len(cfg.AutoDomains.Certs) == 0 {
			DefaultLogger.Warnf("Warning: auto_domains section found in config, but 'certs' map is empty or missing.")
		}
		// All other validations (domains list not empty, key_type validity) are handled by schema
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

# Timeout for ACME challenges (e.g., DNS propagation checks). Default: 10m
# Format: Go duration string (e.g., "5m", "10m30s", "1h")
challenge_timeout: "10m"

# Timeout for HTTP requests made to the ACME server. Default: 30s
# Format: Go duration string (e.g., "30s", "1m")
http_timeout: "30s"

# Storage for acme-dns account credentials is now in a separate JSON file:
# See '<cert_storage_path>/acme-dns-accounts.json'

# Optional section for configuring automatic renewals via the -auto flag.
# If this section is present and -auto is used, the tool will check
# certificates defined here and renew them if they expire within 'graceDays'.
#auto_domains:
#  grace_days: 30 # Renew certs expiring within this many days (default: 30)
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

// validateConfig validates the configuration against the JSON schema.
// It returns nil if the configuration is valid, or an error with validation messages otherwise.
func validateConfig(config []byte) error {
	// Convert YAML to JSON for validation
	var yamlObj interface{}
	if err := yaml.Unmarshal(config, &yamlObj); err != nil {
		return fmt.Errorf("error parsing YAML: %w", err)
	}

	jsonData, err := json.Marshal(yamlObj)
	if err != nil {
		return fmt.Errorf("error converting YAML to JSON: %w", err)
	}

	// Compile the schema
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile([]byte(ConfigSchema))
	if err != nil {
		return fmt.Errorf("schema compilation error: %w", err)
	}

	// Unmarshal JSON data into an interface{} for validation
	var instance interface{}
	if err := json.Unmarshal(jsonData, &instance); err != nil {
		return fmt.Errorf("error parsing JSON for validation: %w", err)
	}

	// Validate the instance against the schema
	result := schema.Validate(instance)
	if !result.IsValid() {
		// Use our custom error formatter for friendly error messages
		return FormatValidationError(result)
	}

	return nil
}
