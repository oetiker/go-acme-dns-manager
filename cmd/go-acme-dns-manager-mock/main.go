//go:build testutils
// +build testutils

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/app"
	"github.com/oetiker/go-acme-dns-manager/pkg/common"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager/test_helpers"
	"github.com/oetiker/go-acme-dns-manager/pkg/manager/test_mocks"
)

// Version information (this will be replaced during build)
var version = "mock-version"

// main creates and runs the mock version of go-acme-dns-manager
// This binary always runs in mock mode with internal mock servers
func main() {
	fmt.Println("🧪 Starting go-acme-dns-manager MOCK VERSION")
	fmt.Println("📡 All ACME operations will be mocked - no real network calls!")

	// Start mock servers
	fmt.Println("🚀 Starting mock ACME-DNS server...")
	mockAcmeDns := test_mocks.NewMockAcmeDnsServer()
	defer func() {
		fmt.Println("🛑 Shutting down mock ACME-DNS server...")
		mockAcmeDns.Close()
	}()

	fmt.Println("🚀 Starting mock ACME server...")
	mockAcme := test_mocks.NewMockAcmeServer()
	defer func() {
		fmt.Println("🛑 Shutting down mock ACME server...")
		mockAcme.Close()
	}()

	fmt.Printf("📍 Mock ACME-DNS server running at: %s\n", mockAcmeDns.GetURL())
	fmt.Printf("📍 Mock ACME server running at: %s/directory\n", mockAcme.GetURL())

	// Replace the default Lego runner with mock implementation
	fmt.Println("🔧 Configuring mock certificate operations...")
	app.DefaultLegoRunner = test_helpers.MockLegoRun

	// Create application with dependency injection
	application := app.NewApplication(version)

	// Setup command line flags
	application.SetupFlags()

	// Parse flags and populate configuration
	application.ParseFlags()

	// Override server URLs to use our mock servers
	if err := application.OverrideServersForMock(mockAcme.GetURL()+"/directory", mockAcmeDns.GetURL()); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to configure mock servers: %v\n", err)
		os.Exit(1)
	}

	// Create context for cancellation/timeout support
	ctx := context.Background()

	// Add overall application timeout (30 minutes max)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	fmt.Println("🎯 Running application with mock infrastructure...")

	// Run the application with enhanced error handling and graceful shutdown
	if err := application.Run(ctx); err != nil {
		handleApplicationError(err)
		os.Exit(1)
	}

	fmt.Println("✅ Mock application completed successfully!")

	// Wait for graceful shutdown if needed
	application.WaitForShutdown()
}

// handleApplicationError provides user-friendly error messages and debugging information
// This is identical to the production version but with mock context
func handleApplicationError(err error) {
	// Check if it's our structured ApplicationError
	if appErr := common.GetApplicationError(err); appErr != nil {
		// This is our structured error - provide detailed information
		fmt.Fprintf(os.Stderr, "❌ Mock Application Error:\n")
		fmt.Fprintf(os.Stderr, "%s\n", appErr.GetDetailedMessage())

		// Provide type-specific guidance for mock version
		switch appErr.Type {
		case common.ErrorTypeConfig:
			fmt.Fprintf(os.Stderr, "\n🔧 Configuration Help (Mock Mode):\n")
			fmt.Fprintf(os.Stderr, "   Use -print-config-template to see a valid template\n")
			fmt.Fprintf(os.Stderr, "   Check file syntax with YAML validators\n")
			fmt.Fprintf(os.Stderr, "   Note: ACME server URLs will be overridden with mock servers\n")
		case common.ErrorTypeNetwork:
			fmt.Fprintf(os.Stderr, "\n🌐 Network Help (Mock Mode):\n")
			fmt.Fprintf(os.Stderr, "   In mock mode, all network calls are simulated\n")
			fmt.Fprintf(os.Stderr, "   This error suggests a problem with the mock infrastructure\n")
		case common.ErrorTypeDNS:
			fmt.Fprintf(os.Stderr, "\n🔍 DNS Help (Mock Mode):\n")
			fmt.Fprintf(os.Stderr, "   DNS operations are mocked - this shouldn't happen\n")
			fmt.Fprintf(os.Stderr, "   Check mock DNS resolver configuration\n")
		case common.ErrorTypeValidation:
			fmt.Fprintf(os.Stderr, "\n✅ Validation Help:\n")
			fmt.Fprintf(os.Stderr, "   Check command line arguments and flags\n")
			fmt.Fprintf(os.Stderr, "   Use -h for usage information\n")
		}
	} else {
		// Generic error handling for non-structured errors
		fmt.Fprintf(os.Stderr, "Mock application error: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "\n💡 For more help, use -h flag or check the documentation.\n")
	fmt.Fprintf(os.Stderr, "🧪 Remember: This is the MOCK version - all operations are simulated!\n")
}
