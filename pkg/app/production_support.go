//go:build !testutils
// +build !testutils

package app

import (
	"github.com/oetiker/go-acme-dns-manager/pkg/manager"
)

// applyMockOverrides is a no-op in the production build
// The mock version will override this with build tags
func (app *Application) applyMockOverrides(cfg *manager.Config) {
	// No-op in production build
}
