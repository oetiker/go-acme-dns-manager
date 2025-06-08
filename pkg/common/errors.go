package common

import (
	"fmt"
	"strings"
)

// ErrorType represents different categories of errors in the application
type ErrorType string

const (
	// ErrorTypeConfig represents configuration-related errors
	ErrorTypeConfig ErrorType = "CONFIG"
	// ErrorTypeNetwork represents network-related errors
	ErrorTypeNetwork ErrorType = "NETWORK"
	// ErrorTypeDNS represents DNS-related errors
	ErrorTypeDNS ErrorType = "DNS"
	// ErrorTypeStorage represents file/storage-related errors
	ErrorTypeStorage ErrorType = "STORAGE"
	// ErrorTypeACME represents ACME protocol errors
	ErrorTypeACME ErrorType = "ACME"
	// ErrorTypeCertificate represents certificate processing errors
	ErrorTypeCertificate ErrorType = "CERTIFICATE"
	// ErrorTypeValidation represents validation errors
	ErrorTypeValidation ErrorType = "VALIDATION"
	// ErrorTypeAuthentication represents authentication errors
	ErrorTypeAuthentication ErrorType = "AUTHENTICATION"
)

// ApplicationError is our custom error type that provides structured error information
type ApplicationError struct {
	Type        ErrorType
	Operation   string // What operation was being performed
	Resource    string // What resource was involved (e.g., file path, domain name)
	Message     string // Human-readable error message
	Underlying  error  // The original error that caused this
	Context     map[string]interface{} // Additional context for debugging
	Suggestions []string // Helpful suggestions for resolving the error
}

// Error implements the error interface
func (e *ApplicationError) Error() string {
	var parts []string

	// Add type and operation context
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("[%s] %s", e.Type, e.Operation))
	} else {
		parts = append(parts, string(e.Type))
	}

	// Add resource context if available
	if e.Resource != "" {
		parts = append(parts, fmt.Sprintf("resource=%s", e.Resource))
	}

	// Add the main message
	parts = append(parts, e.Message)

	// Join all parts
	result := strings.Join(parts, ": ")

	// Add underlying error if present
	if e.Underlying != nil {
		result += fmt.Sprintf(" (cause: %v)", e.Underlying)
	}

	return result
}

// Unwrap returns the underlying error for error chaining
func (e *ApplicationError) Unwrap() error {
	return e.Underlying
}

// IsType checks if the error is of a specific type
func (e *ApplicationError) IsType(errorType ErrorType) bool {
	return e.Type == errorType
}

// AddContext adds additional context to the error
func (e *ApplicationError) AddContext(key string, value interface{}) *ApplicationError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// AddSuggestion adds a helpful suggestion for resolving the error
func (e *ApplicationError) AddSuggestion(suggestion string) *ApplicationError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// GetDetailedMessage returns a detailed error message including context and suggestions
func (e *ApplicationError) GetDetailedMessage() string {
	message := e.Error()

	// Add context if available
	if len(e.Context) > 0 {
		var contextParts []string
		for key, value := range e.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", key, value))
		}
		message += fmt.Sprintf("\nContext: %s", strings.Join(contextParts, ", "))
	}

	// Add suggestions if available
	if len(e.Suggestions) > 0 {
		message += "\nSuggestions:"
		for _, suggestion := range e.Suggestions {
			message += fmt.Sprintf("\n  - %s", suggestion)
		}
	}

	return message
}

// NewApplicationError creates a new application error
func NewApplicationError(errorType ErrorType, operation, message string) *ApplicationError {
	return &ApplicationError{
		Type:      errorType,
		Operation: operation,
		Message:   message,
		Context:   make(map[string]interface{}),
	}
}

// WrapError wraps an existing error with application context
func WrapError(underlying error, errorType ErrorType, operation, message string) *ApplicationError {
	return &ApplicationError{
		Type:       errorType,
		Operation:  operation,
		Message:    message,
		Underlying: underlying,
		Context:    make(map[string]interface{}),
	}
}

// IsApplicationError checks if an error is an ApplicationError
func IsApplicationError(err error) bool {
	_, ok := err.(*ApplicationError)
	return ok
}

// GetApplicationError extracts the ApplicationError from an error chain
func GetApplicationError(err error) *ApplicationError {
	if appErr, ok := err.(*ApplicationError); ok {
		return appErr
	}
	return nil
}

// Common error creation helpers for specific error types

// NewConfigError creates a configuration-related error
func NewConfigError(operation, message string) *ApplicationError {
	return NewApplicationError(ErrorTypeConfig, operation, message).
		AddSuggestion("Check your configuration file syntax and values").
		AddSuggestion("Use -print-config-template to see a valid template")
}

// NewNetworkError creates a network-related error
func NewNetworkError(operation, message string) *ApplicationError {
	return NewApplicationError(ErrorTypeNetwork, operation, message).
		AddSuggestion("Check your network connectivity").
		AddSuggestion("Verify firewall settings and proxy configuration")
}

// NewDNSError creates a DNS-related error
func NewDNSError(operation, message string) *ApplicationError {
	return NewApplicationError(ErrorTypeDNS, operation, message).
		AddSuggestion("Verify DNS server configuration").
		AddSuggestion("Check CNAME record setup")
}

// NewStorageError creates a storage-related error
func NewStorageError(operation, message string) *ApplicationError {
	return NewApplicationError(ErrorTypeStorage, operation, message).
		AddSuggestion("Check file permissions and disk space").
		AddSuggestion("Ensure parent directory exists")
}

// NewACMEError creates an ACME protocol error
func NewACMEError(operation, message string) *ApplicationError {
	return NewApplicationError(ErrorTypeACME, operation, message).
		AddSuggestion("Check ACME server status and connectivity").
		AddSuggestion("Verify account credentials and rate limits")
}

// NewCertificateError creates a certificate processing error
func NewCertificateError(operation, message string) *ApplicationError {
	return NewApplicationError(ErrorTypeCertificate, operation, message).
		AddSuggestion("Check certificate file format and validity").
		AddSuggestion("Verify domain names and certificate chain")
}

// NewValidationError creates a validation error
func NewValidationError(operation, message string) *ApplicationError {
	return NewApplicationError(ErrorTypeValidation, operation, message).
		AddSuggestion("Check input format and values").
		AddSuggestion("Refer to documentation for valid formats")
}
