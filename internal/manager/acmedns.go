package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// RegisterNewAccount interacts with the acme-dns server's /register endpoint.
// It updates the account store with the new account details and saves the store file.
// Exported function
func RegisterNewAccount(cfg *Config, store *accountStore, domain string) (*AcmeDnsAccount, error) {
	registerURL, err := url.JoinPath(cfg.AcmeDnsServer, "/register")
	if err != nil {
		return nil, fmt.Errorf("constructing register URL: %w", err)
	}

	log.Printf("Registering new acme-dns account for %s at %s", domain, registerURL)

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
	defer resp.Body.Close()

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

	// Store the new account details in the account store
	store.SetAccount(domain, newAccount)

	// Save the updated account store file immediately
	if err := store.SaveAccounts(); err != nil {
		// Log the error but potentially continue? Or should this be fatal?
		// For now, log and return the error, as saving is critical.
		log.Printf("Error saving account store after registering %s: %v", domain, err)
		return nil, fmt.Errorf("saving account store after registration: %w", err)
	}

	log.Printf("Successfully registered %s. Account details saved to %s.", domain, store.filePath)
	return &newAccount, nil
}

// PrintRequiredCname prints the CNAME record needed for the user to configure.
// Exported function
func PrintRequiredCname(domain string, fulldomain string) {
	fmt.Println(string(make([]byte, 60))) // Cheap way to print a line
	fmt.Printf("Required DNS CNAME Record:\n")
	fmt.Printf("  _acme-challenge.%s. IN CNAME %s.\n", domain, fulldomain)
	fmt.Println(string(make([]byte, 60)))
}
