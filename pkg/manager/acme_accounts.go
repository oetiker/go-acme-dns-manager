package manager

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/registration"
)

// Simple ACME User struct implementing registration.User
type MyUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *MyUser) GetEmail() string {
	return u.Email
}
func (u *MyUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *MyUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// createOrLoadUser creates a new ACME user or loads an existing one from storage.
func createOrLoadUser(cfg *Config) (*MyUser, error) {
	// Determine storage path relative to config file
	baseStorageDir := cfg.CertStoragePath

	// Extract ACME server hostname from URL to create server-specific directory
	acmeURL, urlErr := url.Parse(cfg.AcmeServer)
	if urlErr != nil {
		return nil, fmt.Errorf("failed to parse ACME server URL: %w", urlErr)
	}

	// Create Lego-style account path structure
	accountsBaseDir := filepath.Join(baseStorageDir, "accounts")
	serverDir := filepath.Join(accountsBaseDir, acmeURL.Host)
	emailDir := filepath.Join(serverDir, cfg.Email)

	// Ensure the directory exists
	if err := os.MkdirAll(emailDir, DirPermissions); err != nil {
		return nil, fmt.Errorf("creating account directory %s: %w", emailDir, err)
	}

	// Set paths for account file and key (using Lego's exact naming conventions)
	accountFilePath := filepath.Join(serverDir, "account.json")

	// Keys are stored in a subdirectory with email as filename
	keysDir := filepath.Join(emailDir, "keys")
	if err := os.MkdirAll(keysDir, DirPermissions); err != nil {
		return nil, fmt.Errorf("creating keys directory %s: %w", keysDir, err)
	}
	keyFilePath := filepath.Join(keysDir, cfg.Email+".key")

	var privateKey crypto.PrivateKey

	// Check if key file exists (in the new location first, then fall back to old location)
	if _, err := os.Stat(keyFilePath); os.IsNotExist(err) {

		// Neither exists, create a new key
		accountKeyType := "ec384" // Always use EC384 for account keys
		DefaultLogger.Infof("Generating new private key (%s) for ACME account", accountKeyType)

		// Generate new key for account (always ec384 for best security/performance)
		var keyErr error
		privateKey, keyErr = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if keyErr != nil {
			return nil, fmt.Errorf("generating private key: %w", keyErr)
		}

		// Save the new key
		keyBytes := certcrypto.PEMEncode(privateKey)
		if writeErr := os.WriteFile(keyFilePath, keyBytes, PrivateKeyPermissions); writeErr != nil {
			return nil, fmt.Errorf("saving private key to %s: %w", keyFilePath, writeErr)
		}
		DefaultLogger.Infof("Saved new private key to %s", keyFilePath)
	} else if err != nil {
		return nil, fmt.Errorf("checking private key file %s: %w", keyFilePath, err)
	} else {
		// Load existing key from the new location
		DefaultLogger.Infof("Loading existing private key from %s", keyFilePath)
		keyBytes, readErr := os.ReadFile(keyFilePath)
		if readErr != nil {
			return nil, fmt.Errorf("reading private key file %s: %w", keyFilePath, readErr)
		}
		var parseErr error
		privateKey, parseErr = certcrypto.ParsePEMPrivateKey(keyBytes)
		if parseErr != nil {
			return nil, fmt.Errorf("parsing private key from %s: %w", keyFilePath, parseErr)
		}
	}

	user := &MyUser{
		Email: cfg.Email,
		key:   privateKey,
	}

	// Load registration info if it exists
	if _, statErr := os.Stat(accountFilePath); statErr == nil {
		DefaultLogger.Infof("Loading existing ACME registration from %s", accountFilePath)
		accountBytes, readErr := os.ReadFile(accountFilePath)
		if readErr != nil {
			return nil, fmt.Errorf("reading account file %s: %w", accountFilePath, readErr)
		}
		jsonErr := json.Unmarshal(accountBytes, &user.Registration)
		if jsonErr != nil {
			return nil, fmt.Errorf("parsing account file %s: %w", accountFilePath, jsonErr)
		}
	} else if !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("checking account file %s: %w", accountFilePath, statErr)
	}

	return user, nil
}

// saveUser saves the user's registration resource.
func saveUser(cfg *Config, user *MyUser) error {
	if user.Registration == nil {
		return fmt.Errorf("cannot save user without registration resource")
	}

	// Extract ACME server hostname from URL for server-specific directory
	acmeURL, urlErr := url.Parse(cfg.AcmeServer)
	if urlErr != nil {
		return fmt.Errorf("failed to parse ACME server URL: %w", urlErr)
	}

	// Create Lego-style account path structure
	accountsBaseDir := filepath.Join(cfg.CertStoragePath, "accounts")
	serverDir := filepath.Join(accountsBaseDir, acmeURL.Host)
	emailDir := filepath.Join(serverDir, cfg.Email)

	// Ensure the directory exists
	if err := os.MkdirAll(emailDir, DirPermissions); err != nil {
		return fmt.Errorf("creating account directory %s: %w", emailDir, err)
	}

	// Set path for account file using Lego's exact naming convention
	accountFilePath := filepath.Join(serverDir, "account.json")

	regBytes, err := json.MarshalIndent(user.Registration, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling registration resource: %w", err)
	}

	err = os.WriteFile(accountFilePath, regBytes, PrivateKeyPermissions)
	if err != nil {
		return fmt.Errorf("writing account file %s: %w", accountFilePath, err)
	}
	DefaultLogger.Infof("Saved ACME registration to %s", accountFilePath)
	return nil
}
