package common

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"
)

// DefaultOperationTimeout is the default timeout for operations
const DefaultOperationTimeout = 30 * time.Second

// DefaultNetworkTimeout is the default timeout for network operations
const DefaultNetworkTimeout = 10 * time.Second

// DefaultDNSLookupTimeout is the default timeout for DNS lookup operations
const DefaultDNSLookupTimeout = 5 * time.Second

// WithTimeout creates a context with a timeout for the operation
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// WithOperationTimeout creates a context with the default operation timeout
func WithOperationTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, DefaultOperationTimeout)
}

// WithNetworkTimeout creates a context with the default network timeout
func WithNetworkTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, DefaultNetworkTimeout)
}

// WithDNSTimeout creates a context with the default DNS timeout
func WithDNSTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, DefaultDNSLookupTimeout)
}

// WithRequestID adds a unique request ID to the context for tracing
func WithRequestID(parent context.Context) context.Context {
	requestID := generateRequestID()
	return context.WithValue(parent, ContextKeyRequestID, requestID)
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		return requestID
	}
	return "unknown"
}

// WithDomain adds domain information to the context
func WithDomain(parent context.Context, domain string) context.Context {
	return context.WithValue(parent, ContextKeyDomain, domain)
}

// GetDomain retrieves the domain from the context
func GetDomain(ctx context.Context) string {
	if domain, ok := ctx.Value(ContextKeyDomain).(string); ok {
		return domain
	}
	return ""
}

// WithOperation adds operation information to the context
func WithOperation(parent context.Context, operation string) context.Context {
	return context.WithValue(parent, ContextKeyOperation, operation)
}

// GetOperation retrieves the operation from the context
func GetOperation(ctx context.Context) string {
	if operation, ok := ctx.Value(ContextKeyOperation).(string); ok {
		return operation
	}
	return ""
}

// IsContextCanceled checks if the context has been canceled or timed out
func IsContextCanceled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// GetContextError returns an appropriate ApplicationError for context cancellation/timeout
func GetContextError(ctx context.Context, operation string) *ApplicationError {
	err := ctx.Err()
	if err == nil {
		return nil
	}

	var errorType ErrorType
	var message string

	switch err {
	case context.Canceled:
		errorType = ErrorTypeValidation
		message = "Operation was canceled"
	case context.DeadlineExceeded:
		errorType = ErrorTypeNetwork
		message = "Operation timed out"
	default:
		errorType = ErrorTypeValidation
		message = "Context error occurred"
	}

	appErr := NewApplicationError(errorType, operation, message).
		AddContext("context_error", err.Error()).
		AddContext("request_id", GetRequestID(ctx))

	if domain := GetDomain(ctx); domain != "" {
		_ = appErr.AddContext("domain", domain)
	}

	// Add appropriate suggestions based on error type
	switch err {
	case context.Canceled:
		_ = appErr.AddSuggestion("Check if the operation was intentionally canceled").
			AddSuggestion("Ensure proper signal handling in your application")
	case context.DeadlineExceeded:
		_ = appErr.AddSuggestion("Increase timeout values if the operation needs more time").
			AddSuggestion("Check network connectivity and server responsiveness").
			AddSuggestion("Consider breaking large operations into smaller chunks")
	}

	return appErr
}

// generateRequestID creates a unique request ID for tracing
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("req_%x", bytes)
}

// CreateOperationContext creates a context for a specific operation with timeout and tracing
func CreateOperationContext(parent context.Context, operation string, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := WithRequestID(parent)
	ctx = WithOperation(ctx, operation)
	return WithTimeout(ctx, timeout)
}

// CreateDomainOperationContext creates a context for domain-specific operations
func CreateDomainOperationContext(parent context.Context, operation, domain string, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := WithRequestID(parent)
	ctx = WithOperation(ctx, operation)
	ctx = WithDomain(ctx, domain)
	return WithTimeout(ctx, timeout)
}
