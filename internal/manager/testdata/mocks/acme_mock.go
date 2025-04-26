package mocks

import (
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509/pkix"
	"encoding/json"
)

// MockAcmeServer simulates a simplified Let's Encrypt ACME server for testing
type MockAcmeServer struct {
	Server       *httptest.Server
	accounts     map[string]map[string]interface{}
	orders       map[string]map[string]interface{}
	challenges   map[string]map[string]interface{}
	certificates map[string][]byte
	mu           sync.RWMutex
}

// NewMockAcmeServer creates and starts a mock ACME server
func NewMockAcmeServer() *MockAcmeServer {
	mock := &MockAcmeServer{
		accounts:     make(map[string]map[string]interface{}),
		orders:       make(map[string]map[string]interface{}),
		challenges:   make(map[string]map[string]interface{}),
		certificates: make(map[string][]byte),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/directory":
			mock.handleDirectory(w, r)
		case r.URL.Path == "/new-account":
			mock.handleNewAccount(w, r)
		case r.URL.Path == "/new-order":
			mock.handleNewOrder(w, r)
		case r.URL.Path == "/challenge":
			mock.handleChallenge(w, r)
		case r.URL.Path == "/finalize-order":
			mock.handleFinalizeOrder(w, r)
		case r.URL.Path == "/certificate":
			mock.handleCertificate(w, r)
		default:
			http.NotFound(w, r)
		}
	}))

	mock.Server = server
	return mock
}

// Close stops the mock server
func (m *MockAcmeServer) Close() {
	m.Server.Close()
}

// GetURL returns the URL of the mock server
func (m *MockAcmeServer) GetURL() string {
	return m.Server.URL
}

// handleDirectory serves the ACME directory
func (m *MockAcmeServer) handleDirectory(w http.ResponseWriter, r *http.Request) {
	directory := map[string]string{
		"newAccount": m.Server.URL + "/new-account",
		"newOrder":   m.Server.URL + "/new-order",
		"revokeCert": m.Server.URL + "/revoke-cert",
		"keyChange":  m.Server.URL + "/key-change",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(directory)
}

// handleNewAccount handles new account registration
func (m *MockAcmeServer) handleNewAccount(w http.ResponseWriter, r *http.Request) {
	// Simplified - just create a dummy account
	accountID := generateRandomString(16)

	m.mu.Lock()
	m.accounts[accountID] = map[string]interface{}{
		"status": "valid",
		"orders": m.Server.URL + "/orders/" + accountID,
	}
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", m.Server.URL+"/acct/"+accountID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(m.accounts[accountID])
}

// handleNewOrder handles new order creation
func (m *MockAcmeServer) handleNewOrder(w http.ResponseWriter, r *http.Request) {
	// Parse the order request (simplified)
	var orderReq struct {
		Identifiers []map[string]string `json:"identifiers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&orderReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Create a mock order
	orderID := generateRandomString(16)
	challenges := []map[string]interface{}{}

	// Create DNS challenges for each domain
	for _, identifier := range orderReq.Identifiers {
		domain := identifier["value"]
		challengeToken := "token-" + generateRandomString(8)
		challengeID := generateRandomString(16)

		challenge := map[string]interface{}{
			"type":   "dns-01",
			"url":    m.Server.URL + "/challenge",
			"token":  challengeToken,
			"status": "pending",
		}

		m.mu.Lock()
		m.challenges[challengeID] = map[string]interface{}{
			"domain":    domain,
			"token":     challengeToken,
			"validated": false,
		}
		m.mu.Unlock()

		challenges = append(challenges, challenge)
	}

	order := map[string]interface{}{
		"status":      "pending",
		"expires":     time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"identifiers": orderReq.Identifiers,
		"challenges":  challenges,
		"finalize":    m.Server.URL + "/finalize-order",
	}

	m.mu.Lock()
	m.orders[orderID] = order
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", m.Server.URL+"/order/"+orderID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

// handleChallenge processes a challenge submission
func (m *MockAcmeServer) handleChallenge(w http.ResponseWriter, r *http.Request) {
	// In a real test, we'd verify the DNS record was set correctly
	// For this mock, we'll just mark the challenge as valid

	challengeID := r.URL.Query().Get("id")
	if challengeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	challenge, exists := m.challenges[challengeID]
	if exists {
		challenge["validated"] = true
	}
	m.mu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status": "valid",
		"type":   "dns-01",
	}
	json.NewEncoder(w).Encode(response)
}

// handleFinalizeOrder creates a certificate for the order
func (m *MockAcmeServer) handleFinalizeOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m.mu.RLock()
	order, exists := m.orders[orderID]
	m.mu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Create a mock certificate
	certPEM, err := generateMockCertificate(order["identifiers"].([]map[string]string))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	certID := generateRandomString(16)
	m.mu.Lock()
	m.certificates[certID] = certPEM
	order["status"] = "valid"
	order["certificate"] = m.Server.URL + "/certificate?id=" + certID
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// handleCertificate serves the mock certificate
func (m *MockAcmeServer) handleCertificate(w http.ResponseWriter, r *http.Request) {
	certID := r.URL.Query().Get("id")
	if certID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m.mu.RLock()
	certPEM, exists := m.certificates[certID]
	m.mu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/pem-certificate-chain")
	w.Write(certPEM)
}

// generateMockCertificate creates a self-signed cert for testing
func generateMockCertificate(identifiers []map[string]string) ([]byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Build the certificate
	notBefore := time.Now()
	notAfter := notBefore.Add(90 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	// Extract domain names from identifiers
	dnsNames := []string{}
	for _, identifier := range identifiers {
		if identifier["type"] == "dns" {
			dnsNames = append(dnsNames, identifier["value"])
		}
	}

	if len(dnsNames) == 0 {
		dnsNames = append(dnsNames, "example.com")
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Mock ACME CA"},
			CommonName:   dnsNames[0],
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	// PEM encode the certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	// PEM encode the private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})

	// Combined PEM (cert + key)
	combinedPEM := append(certPEM, keyPEM...)
	return combinedPEM, nil
}
