package common

import (
	"errors"
	"strings"
	"testing"
)

// TestApplicationError_Error tests the Error() method formatting
func TestApplicationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ApplicationError
		expected string
	}{
		{
			name: "basic error",
			err: &ApplicationError{
				Type:    ErrorTypeConfig,
				Message: "invalid configuration",
			},
			expected: "CONFIG: invalid configuration",
		},
		{
			name: "error with operation",
			err: &ApplicationError{
				Type:      ErrorTypeNetwork,
				Operation: "connect to server",
				Message:   "connection failed",
			},
			expected: "[NETWORK] connect to server: connection failed",
		},
		{
			name: "error with operation and resource",
			err: &ApplicationError{
				Type:      ErrorTypeDNS,
				Operation: "lookup CNAME",
				Resource:  "example.com",
				Message:   "record not found",
			},
			expected: "[DNS] lookup CNAME: resource=example.com: record not found",
		},
		{
			name: "error with underlying cause",
			err: &ApplicationError{
				Type:       ErrorTypeStorage,
				Operation:  "read file",
				Resource:   "/path/to/file",
				Message:    "file operation failed",
				Underlying: errors.New("permission denied"),
			},
			expected: "[STORAGE] read file: resource=/path/to/file: file operation failed (cause: permission denied)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestApplicationError_Unwrap tests error chaining
func TestApplicationError_Unwrap(t *testing.T) {
	underlying := errors.New("original error")
	appErr := &ApplicationError{
		Type:       ErrorTypeConfig,
		Message:    "wrapper error",
		Underlying: underlying,
	}

	if appErr.Unwrap() != underlying {
		t.Errorf("Unwrap() returned %v, want %v", appErr.Unwrap(), underlying)
	}
}

// TestApplicationError_IsType tests error type checking
func TestApplicationError_IsType(t *testing.T) {
	appErr := &ApplicationError{Type: ErrorTypeNetwork}

	if !appErr.IsType(ErrorTypeNetwork) {
		t.Error("IsType(ErrorTypeNetwork) should return true")
	}

	if appErr.IsType(ErrorTypeConfig) {
		t.Error("IsType(ErrorTypeConfig) should return false")
	}
}

// TestApplicationError_AddContext tests context addition
func TestApplicationError_AddContext(t *testing.T) {
	appErr := &ApplicationError{
		Type:    ErrorTypeConfig,
		Message: "test error",
		Context: make(map[string]interface{}),
	}

	_ = appErr.AddContext("key1", "value1").AddContext("key2", 42)

	if appErr.Context["key1"] != "value1" {
		t.Errorf("Context[key1] = %v, want value1", appErr.Context["key1"])
	}

	if appErr.Context["key2"] != 42 {
		t.Errorf("Context[key2] = %v, want 42", appErr.Context["key2"])
	}
}

// TestApplicationError_AddSuggestion tests suggestion addition
func TestApplicationError_AddSuggestion(t *testing.T) {
	appErr := &ApplicationError{
		Type:    ErrorTypeValidation,
		Message: "validation failed",
	}

	_ = appErr.AddSuggestion("Check input format").AddSuggestion("Verify values")

	if len(appErr.Suggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(appErr.Suggestions))
	}

	if appErr.Suggestions[0] != "Check input format" {
		t.Errorf("First suggestion = %v, want 'Check input format'", appErr.Suggestions[0])
	}

	if appErr.Suggestions[1] != "Verify values" {
		t.Errorf("Second suggestion = %v, want 'Verify values'", appErr.Suggestions[1])
	}
}

// TestApplicationError_GetDetailedMessage tests detailed message formatting
func TestApplicationError_GetDetailedMessage(t *testing.T) {
	appErr := &ApplicationError{
		Type:      ErrorTypeACME,
		Operation: "register account",
		Message:   "registration failed",
		Context:   map[string]interface{}{"server": "acme.example.com", "attempts": 3},
	}
	_ = appErr.AddSuggestion("Check server status").AddSuggestion("Verify credentials")

	detailed := appErr.GetDetailedMessage()

	// Check that all components are included
	if !strings.Contains(detailed, "registration failed") {
		t.Error("Detailed message should contain main error message")
	}

	if !strings.Contains(detailed, "server=acme.example.com") {
		t.Error("Detailed message should contain context")
	}

	if !strings.Contains(detailed, "attempts=3") {
		t.Error("Detailed message should contain all context values")
	}

	if !strings.Contains(detailed, "Check server status") {
		t.Error("Detailed message should contain suggestions")
	}

	if !strings.Contains(detailed, "Verify credentials") {
		t.Error("Detailed message should contain all suggestions")
	}
}

// TestNewApplicationError tests error creation
func TestNewApplicationError(t *testing.T) {
	err := NewApplicationError(ErrorTypeCertificate, "parse certificate", "invalid format")

	if err.Type != ErrorTypeCertificate {
		t.Errorf("Type = %v, want %v", err.Type, ErrorTypeCertificate)
	}

	if err.Operation != "parse certificate" {
		t.Errorf("Operation = %v, want 'parse certificate'", err.Operation)
	}

	if err.Message != "invalid format" {
		t.Errorf("Message = %v, want 'invalid format'", err.Message)
	}

	if err.Context == nil {
		t.Error("Context should be initialized")
	}
}

// TestWrapError tests error wrapping
func TestWrapError(t *testing.T) {
	underlying := errors.New("original error")
	wrapped := WrapError(underlying, ErrorTypeStorage, "write file", "failed to write")

	if wrapped.Underlying != underlying {
		t.Errorf("Underlying = %v, want %v", wrapped.Underlying, underlying)
	}

	if wrapped.Type != ErrorTypeStorage {
		t.Errorf("Type = %v, want %v", wrapped.Type, ErrorTypeStorage)
	}

	if wrapped.Operation != "write file" {
		t.Errorf("Operation = %v, want 'write file'", wrapped.Operation)
	}

	if wrapped.Message != "failed to write" {
		t.Errorf("Message = %v, want 'failed to write'", wrapped.Message)
	}
}

// TestIsApplicationError tests error type detection
func TestIsApplicationError(t *testing.T) {
	appErr := &ApplicationError{Type: ErrorTypeConfig}
	genericErr := errors.New("generic error")

	if !IsApplicationError(appErr) {
		t.Error("IsApplicationError should return true for ApplicationError")
	}

	if IsApplicationError(genericErr) {
		t.Error("IsApplicationError should return false for generic error")
	}
}

// TestGetApplicationError tests error extraction
func TestGetApplicationError(t *testing.T) {
	appErr := &ApplicationError{Type: ErrorTypeConfig, Message: "test"}
	genericErr := errors.New("generic error")

	extracted := GetApplicationError(appErr)
	if extracted != appErr {
		t.Errorf("GetApplicationError returned %v, want %v", extracted, appErr)
	}

	extracted = GetApplicationError(genericErr)
	if extracted != nil {
		t.Errorf("GetApplicationError should return nil for generic error, got %v", extracted)
	}
}

// TestErrorHelpers tests the convenience error creation functions
func TestErrorHelpers(t *testing.T) {
	configErr := NewConfigError("load config", "file not found")
	if configErr.Type != ErrorTypeConfig {
		t.Errorf("NewConfigError should create CONFIG type error, got %v", configErr.Type)
	}

	networkErr := NewNetworkError("connect", "timeout")
	if networkErr.Type != ErrorTypeNetwork {
		t.Errorf("NewNetworkError should create NETWORK type error, got %v", networkErr.Type)
	}

	dnsErr := NewDNSError("lookup", "no record")
	if dnsErr.Type != ErrorTypeDNS {
		t.Errorf("NewDNSError should create DNS type error, got %v", dnsErr.Type)
	}

	storageErr := NewStorageError("write", "permission denied")
	if storageErr.Type != ErrorTypeStorage {
		t.Errorf("NewStorageError should create STORAGE type error, got %v", storageErr.Type)
	}

	acmeErr := NewACMEError("register", "rate limited")
	if acmeErr.Type != ErrorTypeACME {
		t.Errorf("NewACMEError should create ACME type error, got %v", acmeErr.Type)
	}

	certErr := NewCertificateError("parse", "invalid format")
	if certErr.Type != ErrorTypeCertificate {
		t.Errorf("NewCertificateError should create CERTIFICATE type error, got %v", certErr.Type)
	}

	validationErr := NewValidationError("check input", "invalid value")
	if validationErr.Type != ErrorTypeValidation {
		t.Errorf("NewValidationError should create VALIDATION type error, got %v", validationErr.Type)
	}
}

// TestErrorChaining tests that errors work with standard Go error handling
func TestErrorChaining(t *testing.T) {
	underlying := errors.New("root cause")
	wrapped := WrapError(underlying, ErrorTypeNetwork, "connect", "connection failed")

	// Test that errors.Is works
	if !errors.Is(wrapped, underlying) {
		t.Error("errors.Is should find the underlying error")
	}

	// Test that errors.As works
	var appErr *ApplicationError
	if !errors.As(wrapped, &appErr) {
		t.Error("errors.As should extract ApplicationError")
	}

	if appErr.Type != ErrorTypeNetwork {
		t.Errorf("Extracted error type = %v, want %v", appErr.Type, ErrorTypeNetwork)
	}
}
