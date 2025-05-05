package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// RegisterNewAccount interacts with the acme-dns server's /register endpoint.
// It updates the account store with the new account details and saves the store file.
// For wildcard domains, it uses the base domain name for registration to maintain consistency.
// Exported function
func RegisterNewAccount(cfg *Config, store *accountStore, domain string) (*AcmeDnsAccount, error) {
	// Extract the base domain for registration purposes
	baseDomain := GetBaseDomain(domain)

	// Check if we already have an account for the base domain
	if account, exists := store.GetAccount(baseDomain); exists && domain != baseDomain {
		// If we're registering a wildcard but already have an account for the base domain,
		// associate the wildcard with the existing account
		store.SetAccount(domain, account)
		DefaultLogger.Infof("Using existing acme-dns account from %s for %s", baseDomain, domain)

		// Since we're sharing the account, we also need to verify the CNAME is valid
		// to prevent the main loop from requesting the same CNAME again
		cnameValid, _ := VerifyCnameRecord(cfg, domain, account.FullDomain)
		if cnameValid {
			DefaultLogger.Infof("Verified that the CNAME for %s is already properly set up", domain)
		}

		return &account, nil
	}

	// Or if we're registering a base domain but already have an account for the wildcard version
	wildcardDomain := "*." + baseDomain
	if account, exists := store.GetAccount(wildcardDomain); exists && domain != wildcardDomain {
		// Associate the base domain with the existing wildcard account
		store.SetAccount(domain, account)
		DefaultLogger.Infof("Using existing acme-dns account from %s for %s", wildcardDomain, domain)

		// Since we're sharing the account, we also need to verify the CNAME is valid
		// to prevent the main loop from requesting the same CNAME again
		cnameValid, _ := VerifyCnameRecord(cfg, domain, account.FullDomain)
		if cnameValid {
			DefaultLogger.Infof("Verified that the CNAME for %s is already properly set up", domain)
		}

		return &account, nil
	}

	registerURL, err := url.JoinPath(cfg.AcmeDnsServer, "/register")
	if err != nil {
		return nil, fmt.Errorf("constructing register URL: %w", err)
	}

	DefaultLogger.Infof("Registering new acme-dns account for %s at %s", domain, registerURL)

	// acme-dns expects an empty JSON object {}
	requestBody := []byte("{}")

	client := &http.Client{Timeout: 30 * time.Second} // Add a reasonable timeout
	req, err := http.NewRequest("POST", registerURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("creating registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(requestBody)))
	req.Header.Set("User-Agent", "go-acme-dns-manager") // Identify our client

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending registration request to %s: %w", registerURL, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log but don't return, we already have a response to process
			DefaultLogger.Errorf("Failed to close response body: %v", closeErr)
		}
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading registration response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated { // 201
		return nil, fmt.Errorf("failed to register at %s: status %d %s, body: %s",
			registerURL, resp.StatusCode, resp.Status, string(bodyBytes))
	}

	var newAccount AcmeDnsAccount
	err = json.Unmarshal(bodyBytes, &newAccount)
	if err != nil {
		return nil, fmt.Errorf("parsing registration response JSON: %w, body: %s", err, string(bodyBytes))
	}

	// Store the new account details in the account store for the requested domain
	store.SetAccount(domain, newAccount)

	// If this is a wildcard domain, also store for the base domain
	// (baseDomain is already defined at the top of the function)
	if domain != baseDomain {
		store.SetAccount(baseDomain, newAccount)
		DefaultLogger.Infof("Also associating account with base domain %s", baseDomain)
	}

	// If this is a base domain, also store for the wildcard version
	// (wildcardDomain is already defined at the top of the function)
	if domain != wildcardDomain {
		store.SetAccount(wildcardDomain, newAccount)
		DefaultLogger.Infof("Also associating account with wildcard domain %s", wildcardDomain)
	}

	// Save the updated account store file immediately
	saveErr := store.SaveAccounts()
	if saveErr != nil {
		// Log the error but potentially continue? Or should this be fatal?
		// For now, log and return the error, as saving is critical.
		DefaultLogger.Errorf("Error saving account store after registering %s: %v", domain, saveErr)
		return nil, fmt.Errorf("saving account store after registration: %w", saveErr)
	}

	DefaultLogger.Infof("Successfully registered %s. Account details saved to %s.", domain, store.filePath)
	return &newAccount, nil
}

// PrintRequiredCname prints the CNAME record needed for the user to configure.
// Exported function
func PrintRequiredCname(domain string, fulldomain string) {
	baseDomain := GetBaseDomain(domain)
	fmt.Println("\nRequired DNS CNAME Record:")
	fmt.Printf("_acme-challenge.%s. IN CNAME %s.\n", baseDomain, fulldomain)
}
