package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/common"
)

// TestApplication_ContextSupport tests that the application properly supports context cancellation
func TestApplication_ContextSupport(t *testing.T) {
	app := NewApplication("test-version")
	app.config.ConfigPath = "/nonexistent/config.yaml"

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	// Try to load configuration - should fail with context error
	_, err := app.LoadConfigurationWithContext(ctx)

	if err == nil {
		t.Fatal("Expected error due to context timeout")
	}

	// Should be our structured error
	appErr := common.GetApplicationError(err)
	if appErr == nil {
		t.Fatal("Expected ApplicationError for context timeout")
	}

	// Check error type (should be network error for timeout)
	if appErr.Type != common.ErrorTypeNetwork {
		t.Errorf("Expected ErrorTypeNetwork for timeout, got %v", appErr.Type)
	}

	// Check that message mentions timeout
	if !strings.Contains(appErr.Message, "timed out") {
		t.Errorf("Expected timeout message, got: %s", appErr.Message)
	}
}

// TestApplication_LoadConfigurationWithContext tests context-aware config loading
func TestApplication_LoadConfigurationWithContext(t *testing.T) {
	app := NewApplication("test-version")
	app.config.ConfigPath = "/nonexistent/config.yaml"

	// Test with valid context
	ctx := context.Background()
	ctx = common.WithRequestID(ctx)

	_, err := app.LoadConfigurationWithContext(ctx)

	// Should fail because file doesn't exist, but not due to context
	if err == nil {
		t.Fatal("Expected error for nonexistent config file")
	}

	appErr := common.GetApplicationError(err)
	if appErr == nil {
		t.Fatal("Expected ApplicationError")
	}

	// Should have request ID in context
	requestID := appErr.Context["request_id"]
	if requestID == nil {
		t.Error("Expected request_id in error context")
	}

	// Should be config error, not context error
	if appErr.Type != common.ErrorTypeConfig {
		t.Errorf("Expected ErrorTypeConfig, got %v", appErr.Type)
	}
}

// TestApplication_GracefulShutdown tests graceful shutdown functionality
func TestApplication_GracefulShutdown(t *testing.T) {
	app := NewApplication("test-version")

	// Test that shutdown works
	app.Shutdown()

	// After shutdown, done channel should be closed
	select {
	case <-app.done:
		// Expected - channel should be closed
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected done channel to be closed after shutdown")
	}
}

// TestApplication_WaitForShutdown tests waiting for shutdown
func TestApplication_WaitForShutdown(t *testing.T) {
	app := NewApplication("test-version")

	// Start waiting in goroutine
	waitDone := make(chan bool)
	go func() {
		app.WaitForShutdown()
		waitDone <- true
	}()

	// Should not complete immediately
	select {
	case <-waitDone:
		t.Error("WaitForShutdown should not complete immediately")
	case <-time.After(50 * time.Millisecond):
		// Expected behavior
	}

	// Now shutdown
	app.Shutdown()

	// Should complete now
	select {
	case <-waitDone:
		// Expected behavior
	case <-time.After(100 * time.Millisecond):
		t.Error("WaitForShutdown should complete after Shutdown()")
	}
}

// TestApplication_ContextPropagation tests that context is properly propagated
func TestApplication_ContextPropagation(t *testing.T) {
	app := NewApplication("test-version")
	app.config.ConfigPath = "/nonexistent/config.yaml"

	// Create context with domain information
	ctx := common.WithDomain(context.Background(), "example.com")
	ctx = common.WithOperation(ctx, "test_operation")

	_, err := app.LoadConfigurationWithContext(ctx)

	if err == nil {
		t.Fatal("Expected error for nonexistent config file")
	}

	appErr := common.GetApplicationError(err)
	if appErr == nil {
		t.Fatal("Expected ApplicationError")
	}

	// Context should be preserved in error
	if appErr.Context["request_id"] == nil {
		t.Error("Expected request_id to be added to context")
	}
}

// TestApplication_ContextWithTimeout tests application behavior with timeouts
func TestApplication_ContextWithTimeout(t *testing.T) {
	// Create a context that times out quickly
	ctx, cancel := common.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	// Any operation should now detect the timeout
	if !common.IsContextCanceled(ctx) {
		t.Error("Context should be canceled after timeout")
	}

	// Get context error should return appropriate error
	ctxErr := common.GetContextError(ctx, "test operation")
	if ctxErr == nil {
		t.Error("Expected context error for timed out context")
		return // Don't try to access ctxErr if it's nil
	}

	if ctxErr.Type != common.ErrorTypeNetwork {
		t.Errorf("Expected ErrorTypeNetwork for timeout, got %v", ctxErr.Type)
	}
}

// TestApplication_SetupGracefulShutdown tests signal handling setup
func TestApplication_SetupGracefulShutdown(t *testing.T) {
	app := NewApplication("test-version")

	// For now, just verify that the app can be created and has the required fields
	if app.done == nil {
		t.Error("Expected done channel to be initialized")
	}
}

// TestApplication_RequestIDGeneration tests that request IDs are properly generated
func TestApplication_RequestIDGeneration(t *testing.T) {
	app := NewApplication("test-version")
	app.config.ConfigPath = "/nonexistent/config.yaml"

	// Load config multiple times to get different request IDs
	ctx1 := common.WithRequestID(context.Background())
	ctx2 := common.WithRequestID(context.Background())

	// Get request IDs
	id1 := common.GetRequestID(ctx1)
	id2 := common.GetRequestID(ctx2)

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

// TestApplication_ContextCancellation tests that operations respect context cancellation
func TestApplication_ContextCancellation(t *testing.T) {
	app := NewApplication("test-version")
	app.config.ConfigPath = "/tmp/test_config.yaml" // Use a path that could exist

	// Create cancelable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// Try to load configuration
	_, err := app.LoadConfigurationWithContext(ctx)

	if err == nil {
		t.Fatal("Expected error for canceled context")
	}

	// Should be context error
	appErr := common.GetApplicationError(err)
	if appErr == nil {
		t.Fatal("Expected ApplicationError")
	}

	if appErr.Type != common.ErrorTypeValidation {
		t.Errorf("Expected ErrorTypeValidation for cancellation, got %v", appErr.Type)
	}

	if !strings.Contains(appErr.Message, "canceled") {
		t.Errorf("Expected cancellation message, got: %s", appErr.Message)
	}
}

// TestApplication_ContextComposition tests that context values are properly composed
func TestApplication_ContextComposition(t *testing.T) {
	app := NewApplication("test-version")
	app.config.ConfigPath = "/nonexistent/config.yaml"

	// Create complex context
	ctx := context.Background()
	ctx = common.WithRequestID(ctx)
	ctx = common.WithOperation(ctx, "load_config")
	ctx = common.WithDomain(ctx, "test.example.com")

	_, err := app.LoadConfigurationWithContext(ctx)

	if err == nil {
		t.Fatal("Expected error for nonexistent config file")
	}

	appErr := common.GetApplicationError(err)
	if appErr == nil {
		t.Fatal("Expected ApplicationError")
	}

	// All context values should be preserved
	if appErr.Context["request_id"] == nil {
		t.Error("Expected request_id in context")
	}

	// Original context values should still be accessible
	originalRequestID := common.GetRequestID(ctx)
	originalOperation := common.GetOperation(ctx)
	originalDomain := common.GetDomain(ctx)

	if originalRequestID == "unknown" {
		t.Error("Request ID should be preserved in original context")
	}

	if originalOperation != "load_config" {
		t.Errorf("Operation should be preserved, got %s", originalOperation)
	}

	if originalDomain != "test.example.com" {
		t.Errorf("Domain should be preserved, got %s", originalDomain)
	}
}
