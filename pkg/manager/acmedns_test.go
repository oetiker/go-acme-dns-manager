package manager

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

// mockHTTPClient implements HTTPClientInterface for testing
type mockHTTPClient struct {
	responses []*http.Response
	errors    []error
	requests  []*http.Request
	callCount int
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Store the request for verification
	m.requests = append(m.requests, req)

	if m.callCount >= len(m.responses) {
		return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
	}

	response := m.responses[m.callCount]
	var err error
	if m.callCount < len(m.errors) {
		err = m.errors[m.callCount]
	}

	m.callCount++
	return response, err
}

// mockLogger implements LoggerInterface for testing
type mockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
}

func (m *mockLogger) Debug(msg string, args ...interface{})             { m.debugMessages = append(m.debugMessages, fmt.Sprintf(msg, args...)) }
func (m *mockLogger) Info(msg string, args ...interface{})              { m.infoMessages = append(m.infoMessages, fmt.Sprintf(msg, args...)) }
func (m *mockLogger) Warn(msg string, args ...interface{})              { m.warnMessages = append(m.warnMessages, fmt.Sprintf(msg, args...)) }
func (m *mockLogger) Error(msg string, args ...interface{})             { m.errorMessages = append(m.errorMessages, fmt.Sprintf(msg, args...)) }
func (m *mockLogger) Debugf(format string, args ...interface{})         { m.debugMessages = append(m.debugMessages, fmt.Sprintf(format, args...)) }
func (m *mockLogger) Infof(format string, args ...interface{})          { m.infoMessages = append(m.infoMessages, fmt.Sprintf(format, args...)) }
func (m *mockLogger) Warnf(format string, args ...interface{})          { m.warnMessages = append(m.warnMessages, fmt.Sprintf(format, args...)) }
func (m *mockLogger) Errorf(format string, args ...interface{})         { m.errorMessages = append(m.errorMessages, fmt.Sprintf(format, args...)) }
func (m *mockLogger) Importantf(format string, args ...interface{})     { m.infoMessages = append(m.infoMessages, fmt.Sprintf(format, args...)) }

// Helper function to create a mock HTTP response
func createMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// Helper function to create a mock ACME DNS account response
func createMockAcmeDnsAccountResponse() string {
	account := AcmeDnsAccount{
		Username:   "test-username-123",
		Password:   "test-password-456",
		FullDomain: "test-subdomain.acmedns.example.com",
		SubDomain:  "test-subdomain",
	}

	jsonBytes, _ := json.Marshal(account)
	return string(jsonBytes)
}

func TestRegisterNewAccountWithDeps_Success(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		AcmeDnsServer: "https://acme-dns.example.com",
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	mockClient := &mockHTTPClient{
		responses: []*http.Response{
			createMockResponse(http.StatusCreated, createMockAcmeDnsAccountResponse()),
		},
		errors: []error{nil},
	}

	mockLog := &mockLogger{}

	// Test registering a new account
	account, err := RegisterNewAccountWithDeps(cfg, store, "example.com", mockLog, mockClient)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if account == nil {
		t.Fatal("Account should not be nil")
	}

	// Verify the account details
	if account.Username != "test-username-123" {
		t.Errorf("Expected username 'test-username-123', got '%s'", account.Username)
	}

	if account.Password != "test-password-456" {
		t.Errorf("Expected password 'test-password-456', got '%s'", account.Password)
	}

	// Verify the request was made correctly
	if len(mockClient.requests) != 1 {
		t.Fatalf("Expected 1 HTTP request, got %d", len(mockClient.requests))
	}

	req := mockClient.requests[0]
	if req.Method != "POST" {
		t.Errorf("Expected POST request, got %s", req.Method)
	}

	expectedURL := "https://acme-dns.example.com/register"
	if req.URL.String() != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, req.URL.String())
	}

	// Verify headers
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", req.Header.Get("Content-Type"))
	}

	if req.Header.Get("User-Agent") != "go-acme-dns-manager" {
		t.Errorf("Expected User-Agent 'go-acme-dns-manager', got '%s'", req.Header.Get("User-Agent"))
	}

	// Verify the account was stored
	storedAccount, exists := store.GetAccount("example.com")
	if !exists {
		t.Error("Account should be stored in account store")
	} else if storedAccount.Username != account.Username {
		t.Errorf("Stored account username doesn't match: expected %s, got %s", account.Username, storedAccount.Username)
	}

	// Verify log messages
	foundRegistrationMessage := false
	for _, msg := range mockLog.infoMessages {
		if strings.Contains(msg, "Registering new acme-dns account for example.com") {
			foundRegistrationMessage = true
			break
		}
	}
	if !foundRegistrationMessage {
		t.Error("Expected registration log message not found")
	}
}

func TestRegisterNewAccountWithDeps_ExistingAccount(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		AcmeDnsServer: "https://acme-dns.example.com",
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	// Pre-populate with an existing account for the base domain
	existingAccount := AcmeDnsAccount{
		Username:   "existing-username",
		Password:   "existing-password",
		FullDomain: "existing-subdomain.acmedns.example.com",
		SubDomain:  "existing-subdomain",
	}
	store.SetAccount("example.com", existingAccount)

	mockClient := &mockHTTPClient{} // No responses needed
	mockLog := &mockLogger{}

	// Test registering for a wildcard domain when base domain account exists
	account, err := RegisterNewAccountWithDeps(cfg, store, "*.example.com", mockLog, mockClient)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return the existing account
	if account.Username != existingAccount.Username {
		t.Errorf("Expected existing username '%s', got '%s'", existingAccount.Username, account.Username)
	}

	// Should not make any HTTP requests
	if len(mockClient.requests) != 0 {
		t.Errorf("Expected 0 HTTP requests, got %d", len(mockClient.requests))
	}

	// Verify log message about using existing account
	foundExistingMessage := false
	for _, msg := range mockLog.infoMessages {
		if strings.Contains(msg, "Using existing acme-dns account from example.com for *.example.com") {
			foundExistingMessage = true
			break
		}
	}
	if !foundExistingMessage {
		t.Error("Expected 'using existing account' log message not found")
	}
}

func TestRegisterNewAccountWithDeps_HTTPError(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		AcmeDnsServer: "https://acme-dns.example.com",
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	mockClient := &mockHTTPClient{
		responses: []*http.Response{
			createMockResponse(http.StatusInternalServerError, "Internal Server Error"),
		},
		errors: []error{nil},
	}

	mockLog := &mockLogger{}

	// Test HTTP error response
	_, err = RegisterNewAccountWithDeps(cfg, store, "example.com", mockLog, mockClient)

	if err == nil {
		t.Fatal("Expected error for HTTP failure")
	}

	if !strings.Contains(err.Error(), "failed to register") {
		t.Errorf("Expected error about registration failure, got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("Expected error to mention status 500, got: %s", err.Error())
	}
}

func TestRegisterNewAccountWithDeps_NetworkError(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		AcmeDnsServer: "https://acme-dns.example.com",
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	mockClient := &mockHTTPClient{
		responses: []*http.Response{nil},
		errors:    []error{fmt.Errorf("network error: connection refused")},
	}

	mockLog := &mockLogger{}

	// Test network error
	_, err = RegisterNewAccountWithDeps(cfg, store, "example.com", mockLog, mockClient)

	if err == nil {
		t.Fatal("Expected error for network failure")
	}

	if !strings.Contains(err.Error(), "sending registration request") {
		t.Errorf("Expected error about sending request, got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("Expected error to mention connection refused, got: %s", err.Error())
	}
}

func TestRegisterNewAccountWithDeps_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		AcmeDnsServer: "https://acme-dns.example.com",
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	mockClient := &mockHTTPClient{
		responses: []*http.Response{
			createMockResponse(http.StatusCreated, "invalid json response"),
		},
		errors: []error{nil},
	}

	mockLog := &mockLogger{}

	// Test invalid JSON response
	_, err = RegisterNewAccountWithDeps(cfg, store, "example.com", mockLog, mockClient)

	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "parsing registration response JSON") {
		t.Errorf("Expected error about JSON parsing, got: %s", err.Error())
	}
}

func TestRegisterNewAccountWithDeps_InvalidURL(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		AcmeDnsServer: ":", // Invalid URL
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	mockClient := &mockHTTPClient{}
	mockLog := &mockLogger{}

	// Test invalid URL construction
	_, err = RegisterNewAccountWithDeps(cfg, store, "example.com", mockLog, mockClient)

	if err == nil {
		t.Fatal("Expected error for invalid URL")
	}

	if !strings.Contains(err.Error(), "constructing register URL") {
		t.Errorf("Expected error about URL construction, got: %s", err.Error())
	}
}

func TestRegisterNewAccountWithDeps_WildcardToBaseMapping(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		AcmeDnsServer: "https://acme-dns.example.com",
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	// Pre-populate with a wildcard account
	wildcardAccount := AcmeDnsAccount{
		Username:   "wildcard-username",
		Password:   "wildcard-password",
		FullDomain: "wildcard-subdomain.acmedns.example.com",
		SubDomain:  "wildcard-subdomain",
	}
	store.SetAccount("*.example.com", wildcardAccount)

	mockClient := &mockHTTPClient{} // No responses needed
	mockLog := &mockLogger{}

	// Test registering for base domain when wildcard account exists
	account, err := RegisterNewAccountWithDeps(cfg, store, "example.com", mockLog, mockClient)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return the existing wildcard account
	if account.Username != wildcardAccount.Username {
		t.Errorf("Expected wildcard username '%s', got '%s'", wildcardAccount.Username, account.Username)
	}

	// Should not make any HTTP requests
	if len(mockClient.requests) != 0 {
		t.Errorf("Expected 0 HTTP requests, got %d", len(mockClient.requests))
	}

	// Verify log message about using existing wildcard account
	foundExistingMessage := false
	for _, msg := range mockLog.infoMessages {
		if strings.Contains(msg, "Using existing acme-dns account from *.example.com for example.com") {
			foundExistingMessage = true
			break
		}
	}
	if !foundExistingMessage {
		t.Error("Expected 'using existing wildcard account' log message not found")
	}
}

func TestRegisterNewAccountWithDeps_AccountAssociation(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		AcmeDnsServer: "https://acme-dns.example.com",
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	mockClient := &mockHTTPClient{
		responses: []*http.Response{
			createMockResponse(http.StatusCreated, createMockAcmeDnsAccountResponse()),
		},
		errors: []error{nil},
	}

	mockLog := &mockLogger{}

	// Test registering a wildcard domain
	account, err := RegisterNewAccountWithDeps(cfg, store, "*.example.com", mockLog, mockClient)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify the account is associated with both wildcard and base domain
	wildcardAccount, wildcardExists := store.GetAccount("*.example.com")
	baseAccount, baseExists := store.GetAccount("example.com")

	if !wildcardExists {
		t.Error("Wildcard account should exist in store")
	}
	if !baseExists {
		t.Error("Base domain account should exist in store")
	}

	if wildcardAccount.Username != baseAccount.Username {
		t.Error("Wildcard and base domain should share the same account")
	}

	if account.Username != wildcardAccount.Username {
		t.Error("Returned account should match stored account")
	}
}

func TestRegisterNewAccount_WrapperFunction(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		AcmeDnsServer: "https://acme-dns.example.com",
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	// Pre-populate with an existing account to avoid HTTP calls
	existingAccount := AcmeDnsAccount{
		Username:   "existing-username",
		Password:   "existing-password",
		FullDomain: "existing-subdomain.acmedns.example.com",
		SubDomain:  "existing-subdomain",
	}
	store.SetAccount("example.com", existingAccount)

	// Test the wrapper function
	account, err := RegisterNewAccount(cfg, store, "*.example.com")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if account.Username != existingAccount.Username {
		t.Errorf("Expected existing username '%s', got '%s'", existingAccount.Username, account.Username)
	}
}

func TestRegisterNewAccountWithLogger_WrapperFunction(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		AcmeDnsServer: "https://acme-dns.example.com",
	}

	store, err := NewAccountStore(filepath.Join(tmpDir, "accounts.json"))
	if err != nil {
		t.Fatalf("Failed to create account store: %v", err)
	}

	// Pre-populate with an existing account to avoid HTTP calls
	existingAccount := AcmeDnsAccount{
		Username:   "existing-username",
		Password:   "existing-password",
		FullDomain: "existing-subdomain.acmedns.example.com",
		SubDomain:  "existing-subdomain",
	}
	store.SetAccount("example.com", existingAccount)

	mockLog := &mockLogger{}

	// Test the wrapper function with logger
	account, err := RegisterNewAccountWithLogger(cfg, store, "*.example.com", mockLog)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if account.Username != existingAccount.Username {
		t.Errorf("Expected existing username '%s', got '%s'", existingAccount.Username, account.Username)
	}
}


// Benchmark for account registration performance
func BenchmarkRegisterNewAccountWithDeps(b *testing.B) {
	tmpDir := b.TempDir()

	cfg := &Config{
		AcmeDnsServer: "https://acme-dns.example.com",
	}

	mockLog := &mockLogger{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store, _ := NewAccountStore(filepath.Join(tmpDir, fmt.Sprintf("accounts-%d.json", i)))

		// Pre-populate to avoid HTTP calls in benchmark
		existingAccount := AcmeDnsAccount{
			Username:   "bench-username",
			Password:   "bench-password",
			FullDomain: "bench-subdomain.acmedns.example.com",
			SubDomain:  "bench-subdomain",
		}
		store.SetAccount("example.com", existingAccount)

		mockClient := &mockHTTPClient{}

		_, _ = RegisterNewAccountWithDeps(cfg, store, "*.example.com", mockLog, mockClient)
	}
}
