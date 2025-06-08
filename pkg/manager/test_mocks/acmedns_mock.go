//go:build testutils
// +build testutils

package test_mocks

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
)

// MockAcmeDnsServer simulates an ACME DNS server for testing
type MockAcmeDnsServer struct {
	Server    *httptest.Server
	accounts  map[string]map[string]interface{}
	recordsMu sync.RWMutex
	records   map[string]string // Domain -> TXT record
}

// NewMockAcmeDnsServer creates and starts a mock ACME DNS server
func NewMockAcmeDnsServer() *MockAcmeDnsServer {
	mock := &MockAcmeDnsServer{
		accounts: make(map[string]map[string]interface{}),
		records:  make(map[string]string),
	}

	// Create an HTTP test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/register":
			// Handle registration request
			mock.handleRegister(w, r)
		case "/update":
			// Handle DNS record update request
			mock.handleUpdate(w, r)
		default:
			http.NotFound(w, r)
		}
	}))

	mock.Server = server
	return mock
}

// Close stops the mock server
func (m *MockAcmeDnsServer) Close() {
	m.Server.Close()
}

// GetURL returns the URL of the mock server
func (m *MockAcmeDnsServer) GetURL() string {
	return m.Server.URL
}

// handleRegister handles the /register endpoint
func (m *MockAcmeDnsServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Generate a mock account
	username := generateRandomString(8)
	password := generateRandomString(16)
	subdomain := generateRandomString(10)
	fulldomain := subdomain + ".acme-dns.mock.test"

	account := map[string]interface{}{
		"username":   username,
		"password":   password,
		"subdomain":  subdomain,
		"fulldomain": fulldomain,
		"allowfrom":  []string{},
	}

	// Store the account
	m.accounts[username] = account

	// Return the account data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(account); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// handleUpdate handles the /update endpoint
func (m *MockAcmeDnsServer) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse auth from basic auth header
	username, password, ok := r.BasicAuth()
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Check credentials
	account, exists := m.accounts[username]
	if !exists || account["password"] != password {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Parse TXT record from request
	var updateReq struct {
		Subdomain string `json:"subdomain"`
		Txt       string `json:"txt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Store the record
	domain := account["fulldomain"].(string)
	m.recordsMu.Lock()
	m.records[domain] = updateReq.Txt
	m.recordsMu.Unlock()

	// Success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"txt": updateReq.Txt}); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// GetTXTRecord returns the stored TXT record for a domain
func (m *MockAcmeDnsServer) GetTXTRecord(domain string) (string, bool) {
	m.recordsMu.RLock()
	defer m.recordsMu.RUnlock()
	record, exists := m.records[domain]
	return record, exists
}

// Helper function to generate random strings
func generateRandomString(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
