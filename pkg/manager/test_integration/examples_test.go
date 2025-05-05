package test_integration

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/oetiker/go-acme-dns-manager/pkg/manager/test_mocks"
)

// TestMockServers demonstrates how to use both mock servers together
func TestMockServers(t *testing.T) {
	// Skip in regular test runs
	if testing.Short() {
		t.Skip("Skipping mock server example test in short mode")
	}

	// Start the mock ACME DNS server
	mockDNS := test_mocks.NewMockAcmeDnsServer()
	defer mockDNS.Close()

	// Start the mock ACME server
	mockACME := test_mocks.NewMockAcmeServer()
	defer mockACME.Close()

	// Example 1: Test ACME DNS server registration
	t.Run("AcmeDNS_Registration", func(t *testing.T) {
		// Make a registration request
		resp, err := http.Post(mockDNS.GetURL()+"/register", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Warning: Failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		// Read and log response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		t.Logf("Registration response: %s", body)
	})

	// Example 2: Test ACME server directory
	t.Run("ACME_Directory", func(t *testing.T) {
		resp, err := http.Get(mockACME.GetURL() + "/directory")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Warning: Failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Read and log response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		t.Logf("Directory response: %s", body)
	})

	// Example 3: Waiting for mock servers
	t.Run("Server_Availability", func(t *testing.T) {
		// This test demonstrates how to ensure both mock servers are up and running
		deadline := time.Now().Add(5 * time.Second)

		for time.Now().Before(deadline) {
			// Try ACME DNS server
			resp1, err1 := http.Get(mockDNS.GetURL() + "/health")

			// Try ACME server
			resp2, err2 := http.Get(mockACME.GetURL() + "/directory")

			// Check if both are available
			if err1 == nil && err2 == nil {
				if resp1.StatusCode == http.StatusNotFound && resp2.StatusCode == http.StatusOK {
					// Success - ACME DNS returns 404 for unimplemented /health endpoint
					// and ACME returns 200 for /directory
					if err := resp1.Body.Close(); err != nil {
						t.Logf("Warning: Failed to close response body: %v", err)
					}
					if err := resp2.Body.Close(); err != nil {
						t.Logf("Warning: Failed to close response body: %v", err)
					}
					return
				}
				if resp1.Body != nil {
					if err := resp1.Body.Close(); err != nil {
						t.Logf("Warning: Failed to close response body: %v", err)
					}
				}
				if resp2.Body != nil {
					if err := resp2.Body.Close(); err != nil {
						t.Logf("Warning: Failed to close response body: %v", err)
					}
				}
			}

			time.Sleep(500 * time.Millisecond)
		}

		t.Fatal("Timeout waiting for mock servers to be available")
	})
}
