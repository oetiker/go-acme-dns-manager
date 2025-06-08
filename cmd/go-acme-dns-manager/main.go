package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/app"
	"github.com/oetiker/go-acme-dns-manager/pkg/common"
)

// Version information (this will be replaced during build)
var version = "local-version"

// main demonstrates the new, clean application structure with context support
// This replaces the 537-line monolithic main() function with focused, testable components
func main() {
	// Create application with dependency injection
	application := app.NewApplication(version)

	// Setup command line flags
	application.SetupFlags()

	// Parse flags and populate configuration
	application.ParseFlags()

	// Create context for cancellation/timeout support
	ctx := context.Background()

	// Add overall application timeout (30 minutes max)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	// Run the application with enhanced error handling and graceful shutdown
	if err := application.Run(ctx); err != nil {
		handleApplicationError(err)
		os.Exit(1)
	}

	// Wait for graceful shutdown if needed
	application.WaitForShutdown()
}

// handleApplicationError provides user-friendly error messages and debugging information
func handleApplicationError(err error) {
	// Check if it's our structured ApplicationError
	if appErr := common.GetApplicationError(err); appErr != nil {
		// This is our structured error - provide detailed information
		fmt.Fprintf(os.Stderr, "❌ Application Error:\n")
		fmt.Fprintf(os.Stderr, "%s\n", appErr.GetDetailedMessage())

		// Provide type-specific guidance
		switch appErr.Type {
		case common.ErrorTypeConfig:
			fmt.Fprintf(os.Stderr, "\n🔧 Configuration Help:\n")
			fmt.Fprintf(os.Stderr, "   Use -print-config-template to see a valid template\n")
			fmt.Fprintf(os.Stderr, "   Check file syntax with YAML validators\n")
		case common.ErrorTypeNetwork:
			fmt.Fprintf(os.Stderr, "\n🌐 Network Help:\n")
			fmt.Fprintf(os.Stderr, "   Check firewall settings and proxy configuration\n")
			fmt.Fprintf(os.Stderr, "   Verify server URLs are accessible\n")
		case common.ErrorTypeDNS:
			fmt.Fprintf(os.Stderr, "\n🔍 DNS Help:\n")
			fmt.Fprintf(os.Stderr, "   Use 'dig' or 'nslookup' to verify DNS records\n")
			fmt.Fprintf(os.Stderr, "   Check CNAME record configuration\n")
		case common.ErrorTypeValidation:
			fmt.Fprintf(os.Stderr, "\n✅ Validation Help:\n")
			fmt.Fprintf(os.Stderr, "   Check command line arguments and flags\n")
			fmt.Fprintf(os.Stderr, "   Use -h for usage information\n")
		}
	} else {
		// Generic error handling for non-structured errors
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "\n💡 For more help, use -h flag or check the documentation.\n")
}

// This demonstrates how the 537-line main() function becomes just 25 lines
// while being much more testable, maintainable, and focused
