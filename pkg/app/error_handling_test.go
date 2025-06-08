package app

import (
	"strings"
	"testing"

	"github.com/oetiker/go-acme-dns-manager/pkg/common"
)

// TestApplicationError_StructuredErrors demonstrates how our new error handling works
func TestApplicationError_StructuredErrors(t *testing.T) {
	app := NewApplication("test")
	app.config.ConfigPath = "/nonexistent/config.yaml"

	// This should trigger our enhanced config error
	_, err := app.LoadConfiguration()

	if err == nil {
		t.Fatal("Expected error for nonexistent config file")
	}

	// Check that it's our structured error
	if !common.IsApplicationError(err) {
		t.Errorf("Expected ApplicationError, got %T", err)
	}

	appErr := common.GetApplicationError(err)
	if appErr == nil {
		t.Fatal("Should be able to extract ApplicationError")
	}

	// Verify error structure
	if appErr.Type != common.ErrorTypeConfig {
		t.Errorf("Expected ErrorTypeConfig, got %v", appErr.Type)
	}

	if appErr.Operation != "locate config file" {
		t.Errorf("Expected operation 'locate config file', got '%v'", appErr.Operation)
	}

	// Check that context was added
	if appErr.Context["config_path"] != "/nonexistent/config.yaml" {
		t.Errorf("Expected config_path context, got %v", appErr.Context["config_path"])
	}

	// Check that suggestions were added
	if len(appErr.Suggestions) == 0 {
		t.Error("Expected suggestions to be added")
	}

	// Verify suggestions contain helpful text
	foundTemplate := false
	for _, suggestion := range appErr.Suggestions {
		if strings.Contains(suggestion, "print-config-template") {
			foundTemplate = true
			break
		}
	}
	if !foundTemplate {
		t.Error("Expected suggestion about config template")
	}
}

// TestValidateModeError_StructuredErrors tests validation error structure
func TestValidateModeError_StructuredErrors(t *testing.T) {
	app := NewApplication("test")
	app.config.AutoMode = true

	// Try to validate with both auto mode and manual args - should error
	err := app.ValidateModeWithArgs([]string{"example.com"})

	if err == nil {
		t.Fatal("Expected validation error")
	}

	// Check that it's our structured error
	appErr := common.GetApplicationError(err)
	if appErr == nil {
		t.Fatal("Should be able to extract ApplicationError")
	}

	// Verify error structure
	if appErr.Type != common.ErrorTypeValidation {
		t.Errorf("Expected ErrorTypeValidation, got %v", appErr.Type)
	}

	if appErr.Operation != "validate operation mode" {
		t.Errorf("Expected operation 'validate operation mode', got '%v'", appErr.Operation)
	}

	// Check context
	if appErr.Context["auto_mode"] != true {
		t.Errorf("Expected auto_mode=true in context, got %v", appErr.Context["auto_mode"])
	}

	if appErr.Context["manual_args_count"] != 1 {
		t.Errorf("Expected manual_args_count=1 in context, got %v", appErr.Context["manual_args_count"])
	}

	// Check suggestions
	if len(appErr.Suggestions) < 2 {
		t.Errorf("Expected at least 2 suggestions, got %d", len(appErr.Suggestions))
	}
}

// TestErrorMessage_Formatting tests that error messages are well-formatted
func TestErrorMessage_Formatting(t *testing.T) {
	app := NewApplication("test")
	app.config.AutoMode = false

	// No mode specified error
	err := app.ValidateModeWithArgs([]string{})
	appErr := common.GetApplicationError(err)

	// Test basic error message
	basicMsg := appErr.Error()
	if !strings.Contains(basicMsg, "[VALIDATION]") {
		t.Error("Basic error message should contain error type")
	}

	if !strings.Contains(basicMsg, "validate operation mode") {
		t.Error("Basic error message should contain operation")
	}

	// Test detailed error message
	detailedMsg := appErr.GetDetailedMessage()
	if !strings.Contains(detailedMsg, "Context:") {
		t.Error("Detailed message should contain context section")
	}

	if !strings.Contains(detailedMsg, "Suggestions:") {
		t.Error("Detailed message should contain suggestions section")
	}

	if !strings.Contains(detailedMsg, "auto_mode=false") {
		t.Error("Detailed message should show context values")
	}

	if !strings.Contains(detailedMsg, "Use -auto flag") {
		t.Error("Detailed message should contain helpful suggestions")
	}
}

// TestErrorChaining_WithStandardLibrary tests that our errors work with Go's error handling
func TestErrorChaining_WithStandardLibrary(t *testing.T) {
	// Create a wrapped error
	originalErr := common.NewConfigError("test operation", "test message")
	wrappedErr := common.WrapError(originalErr, common.ErrorTypeNetwork, "network op", "network failed")

	// Test that error chains work as expected
	if !common.IsApplicationError(wrappedErr) {
		t.Error("Wrapped error should still be detected as ApplicationError")
	}

	// Test unwrapping
	if wrappedErr.Unwrap() != originalErr {
		t.Error("Unwrap should return the original error")
	}

	// Test that we can extract the most recent error in the chain
	appErr := common.GetApplicationError(wrappedErr)
	if appErr.Type != common.ErrorTypeNetwork {
		t.Errorf("Should extract the wrapping error, got type %v", appErr.Type)
	}
}

// This demonstrates the benefits of structured error handling:
// 1. Errors contain context and debugging information
// 2. Error types enable different handling strategies
// 3. Suggestions provide actionable help to users
// 4. All errors are testable and verifiable
// 5. Compatible with standard Go error handling patterns
