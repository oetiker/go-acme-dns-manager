//go:build testutils
// +build testutils

package app

import (
	"fmt"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// mockServers holds the override URLs for mock testing
type mockServers struct {
	acmeURL    string
	acmeDnsURL string
}

// mockServerOverrides stores the override URLs
var mockServerOverrides *mockServers

// OverrideServersForMock configures the application to use mock server URLs
// This method only exists when built with the testutils tag
func (app *Application) OverrideServersForMock(acmeURL, acmeDnsURL string) error {
	// Store the override URLs
	mockServerOverrides = &mockServers{
		acmeURL:    acmeURL,
		acmeDnsURL: acmeDnsURL,
	}

	if app.logger != nil {
		app.logger.Infof("🧪 Mock mode: ACME server override set to %s", acmeURL)
		app.logger.Infof("🧪 Mock mode: ACME-DNS server override set to %s", acmeDnsURL)
	} else {
		// Logger not set up yet, use fmt for early setup
		fmt.Printf("🧪 Mock mode: ACME server override set to %s\n", acmeURL)
		fmt.Printf("🧪 Mock mode: ACME-DNS server override set to %s\n", acmeDnsURL)
	}

	return nil
}

// applyMockOverrides modifies the manager configuration to use mock servers
// This method overrides the no-op version in the production build
func (app *Application) applyMockOverrides(cfg *manager.Config) {
	if mockServerOverrides != nil {
		app.logger.Infof("🧪 Overriding ACME server: %s -> %s", cfg.AcmeServer, mockServerOverrides.acmeURL)
		app.logger.Infof("🧪 Overriding ACME-DNS server: %s -> %s", cfg.AcmeDnsServer, mockServerOverrides.acmeDnsURL)

		cfg.AcmeServer = mockServerOverrides.acmeURL
		cfg.AcmeDnsServer = mockServerOverrides.acmeDnsURL
	}
}
