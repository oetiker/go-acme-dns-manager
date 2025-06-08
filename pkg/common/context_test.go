package common

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestWithTimeout tests basic timeout functionality
func TestWithTimeout(t *testing.T) {
	parent := context.Background()
	timeout := 100 * time.Millisecond

	ctx, cancel := WithTimeout(parent, timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		t.Error("Context should not be done immediately")
	default:
		// Expected behavior
	}

	time.Sleep(150 * time.Millisecond)

	select {
	case <-ctx.Done():
		// Expected behavior
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
		}
	default:
		t.Error("Context should be done after timeout")
	}
}

// TestDefaultTimeouts tests the default timeout constants
func TestDefaultTimeouts(t *testing.T) {
	if DefaultOperationTimeout != 30*time.Second {
		t.Errorf("Expected DefaultOperationTimeout to be 30s, got %v", DefaultOperationTimeout)
	}

	if DefaultNetworkTimeout != 10*time.Second {
		t.Errorf("Expected DefaultNetworkTimeout to be 10s, got %v", DefaultNetworkTimeout)
	}

	if DefaultDNSLookupTimeout != 5*time.Second {
		t.Errorf("Expected DefaultDNSLookupTimeout to be 5s, got %v", DefaultDNSLookupTimeout)
	}
}

// TestWithRequestID tests request ID functionality
func TestWithRequestID(t *testing.T) {
	parent := context.Background()
	ctx := WithRequestID(parent)

	requestID := GetRequestID(ctx)
	if requestID == "" {
		t.Error("Expected non-empty request ID")
	}

	if requestID == "unknown" {
		t.Error("Expected actual request ID, got 'unknown'")
	}

	if !strings.HasPrefix(requestID, "req_") {
		t.Errorf("Expected request ID to start with 'req_', got %s", requestID)
	}
}

// TestGetRequestIDFromEmptyContext tests request ID retrieval from context without ID
func TestGetRequestIDFromEmptyContext(t *testing.T) {
	ctx := context.Background()
	requestID := GetRequestID(ctx)

	if requestID != "unknown" {
		t.Errorf("Expected 'unknown' for empty context, got %s", requestID)
	}
}

// TestWithDomain tests domain context functionality
func TestWithDomain(t *testing.T) {
	parent := context.Background()
	domain := "example.com"

	ctx := WithDomain(parent, domain)
	retrievedDomain := GetDomain(ctx)

	if retrievedDomain != domain {
		t.Errorf("Expected domain %s, got %s", domain, retrievedDomain)
	}
}

// TestGetDomainFromEmptyContext tests domain retrieval from context without domain
func TestGetDomainFromEmptyContext(t *testing.T) {
	ctx := context.Background()
	domain := GetDomain(ctx)

	if domain != "" {
		t.Errorf("Expected empty string for empty context, got %s", domain)
	}
}

// TestWithOperation tests operation context functionality
func TestWithOperation(t *testing.T) {
	parent := context.Background()
	operation := "test_operation"

	ctx := WithOperation(parent, operation)
	retrievedOperation := GetOperation(ctx)

	if retrievedOperation != operation {
		t.Errorf("Expected operation %s, got %s", operation, retrievedOperation)
	}
}

// TestGetOperationFromEmptyContext tests operation retrieval from context without operation
func TestGetOperationFromEmptyContext(t *testing.T) {
	ctx := context.Background()
	operation := GetOperation(ctx)

	if operation != "" {
		t.Errorf("Expected empty string for empty context, got %s", operation)
	}
}

// TestIsContextCanceled tests context cancellation detection
func TestIsContextCanceled(t *testing.T) {
	// Test non-canceled context
	ctx := context.Background()
	if IsContextCanceled(ctx) {
		t.Error("Background context should not be canceled")
	}

	// Test canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if !IsContextCanceled(ctx) {
		t.Error("Canceled context should be detected as canceled")
	}

	// Test timed out context
	ctx, cancel2 := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel2()

	time.Sleep(10 * time.Millisecond)

	if !IsContextCanceled(ctx) {
		t.Error("Timed out context should be detected as canceled")
	}
}

// TestGetContextError tests context error conversion
func TestGetContextError(t *testing.T) {
	operation := "test_operation"

	// Test non-error context
	ctx := context.Background()
	err := GetContextError(ctx, operation)
	if err != nil {
		t.Errorf("Expected nil error for good context, got %v", err)
	}

	// Test canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = GetContextError(ctx, operation)
	if err == nil {
		t.Error("Expected error for canceled context")
		return // Don't try to access err if it's nil
	}

	if err.Type != ErrorTypeValidation {
		t.Errorf("Expected ErrorTypeValidation for canceled context, got %v", err.Type)
	}

	if !strings.Contains(err.Message, "canceled") {
		t.Errorf("Expected canceled message, got %s", err.Message)
	}

	// Test timed out context
	ctx, cancel2 := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel2()

	time.Sleep(10 * time.Millisecond)

	err = GetContextError(ctx, operation)
	if err == nil {
		t.Error("Expected error for timed out context")
		return // Don't try to access err if it's nil
	}

	if err.Type != ErrorTypeNetwork {
		t.Errorf("Expected ErrorTypeNetwork for timeout, got %v", err.Type)
	}

	if !strings.Contains(err.Message, "timed out") {
		t.Errorf("Expected timeout message, got %s", err.Message)
	}
}

// TestCreateOperationContext tests operation context creation
func TestCreateOperationContext(t *testing.T) {
	parent := context.Background()
	operation := "test_op"
	timeout := 100 * time.Millisecond

	ctx, cancel := CreateOperationContext(parent, operation, timeout)
	defer cancel()

	// Check that operation is set
	if GetOperation(ctx) != operation {
		t.Errorf("Expected operation %s, got %s", operation, GetOperation(ctx))
	}

	// Check that request ID is set
	if GetRequestID(ctx) == "unknown" {
		t.Error("Expected request ID to be set")
	}

	// Check that timeout works
	time.Sleep(150 * time.Millisecond)
	if !IsContextCanceled(ctx) {
		t.Error("Context should be canceled after timeout")
	}
}

// TestCreateDomainOperationContext tests domain operation context creation
func TestCreateDomainOperationContext(t *testing.T) {
	parent := context.Background()
	operation := "test_op"
	domain := "example.com"
	timeout := 100 * time.Millisecond

	ctx, cancel := CreateDomainOperationContext(parent, operation, domain, timeout)
	defer cancel()

	// Check that operation is set
	if GetOperation(ctx) != operation {
		t.Errorf("Expected operation %s, got %s", operation, GetOperation(ctx))
	}

	// Check that domain is set
	if GetDomain(ctx) != domain {
		t.Errorf("Expected domain %s, got %s", domain, GetDomain(ctx))
	}

	// Check that request ID is set
	if GetRequestID(ctx) == "unknown" {
		t.Error("Expected request ID to be set")
	}

	// Check that timeout works
	time.Sleep(150 * time.Millisecond)
	if !IsContextCanceled(ctx) {
		t.Error("Context should be canceled after timeout")
	}
}

// TestGenerateRequestID tests request ID generation
func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == id2 {
		t.Error("Request IDs should be unique")
	}

	if !strings.HasPrefix(id1, "req_") {
		t.Errorf("Request ID should start with 'req_', got %s", id1)
	}

	if !strings.HasPrefix(id2, "req_") {
		t.Errorf("Request ID should start with 'req_', got %s", id2)
	}
}

// TestContextKeyConstants tests that context keys are defined
func TestContextKeyConstants(t *testing.T) {
	if ContextKeyTimeout == "" {
		t.Error("ContextKeyTimeout should not be empty")
	}

	if ContextKeyRequestID == "" {
		t.Error("ContextKeyRequestID should not be empty")
	}

	if ContextKeyDomain == "" {
		t.Error("ContextKeyDomain should not be empty")
	}

	if ContextKeyOperation == "" {
		t.Error("ContextKeyOperation should not be empty")
	}

	// Ensure keys are unique
	keys := []ContextKey{ContextKeyTimeout, ContextKeyRequestID, ContextKeyDomain, ContextKeyOperation}
	for i, key1 := range keys {
		for j, key2 := range keys {
			if i != j && key1 == key2 {
				t.Errorf("Context keys should be unique, but %v == %v", key1, key2)
			}
		}
	}
}

// TestContextComposition tests that context values can be composed together
func TestContextComposition(t *testing.T) {
	parent := context.Background()

	// Build up context with multiple values
	ctx := WithRequestID(parent)
	ctx = WithOperation(ctx, "complex_operation")
	ctx = WithDomain(ctx, "test.example.com")

	// Verify all values are present
	requestID := GetRequestID(ctx)
	operation := GetOperation(ctx)
	domain := GetDomain(ctx)

	if requestID == "unknown" {
		t.Error("Request ID should be set")
	}

	if operation != "complex_operation" {
		t.Errorf("Expected operation 'complex_operation', got %s", operation)
	}

	if domain != "test.example.com" {
		t.Errorf("Expected domain 'test.example.com', got %s", domain)
	}
}
