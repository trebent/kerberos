package security

import (
	"fmt"
	"net/http"
	"testing"

	lib "github.com/trebent/kerberos/test/lib"
)

// ---- Admin API ----

// TestAdminAPITLS verifies that the admin API is reachable over TLS.
// A 401 response confirms the TLS handshake completed and the server is up.
func TestAdminAPITLS(t *testing.T) {
	t.Parallel()

	resp, err := lib.TLSClient(t, certDir).Get(fmt.Sprintf("https://localhost:%d/api/admin/flow", lib.GetAdminPort()))
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// TestAdminAPIPlainHTTP verifies that the admin API rejects plain HTTP connections.
func TestAdminAPIPlainHTTP(t *testing.T) {
	t.Parallel()

	resp, err := lib.PlainClient().Get(fmt.Sprintf("http://localhost:%d/api/admin/flow", lib.GetAdminPort()))
	if err != nil {
		t.Fatalf("Unexpected error when sending plain HTTP request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request when sending plain HTTP to a TLS-only port, got %d", resp.StatusCode)
	}
}

// ---- Gateway API ----

// TestGWAPImTLSEcho verifies that the gateway API is reachable over TLS towards an mTLS enabled backend.
func TestGWAPImTLSEcho(t *testing.T) {
	t.Parallel()

	resp, err := lib.TLSClient(t, certDir).Get(fmt.Sprintf("https://localhost:%d/gw/backend/mtls-echo/hi", lib.GetPort()))
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

// TestGWAPITLSEcho verifies that the gateway API is reachable over TLS towards a TLS enabled backend.
func TestGWAPITLSEcho(t *testing.T) {
	t.Parallel()

	resp, err := lib.TLSClient(t, certDir).Get(fmt.Sprintf("https://localhost:%d/gw/backend/tls-echo/hi", lib.GetPort()))
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

// TestGWAPIPlainHTTP verifies that the gateway API rejects plain HTTP connections.
func TestGWAPIPlainHTTP(t *testing.T) {
	t.Parallel()

	resp, err := lib.PlainClient().Get(fmt.Sprintf("http://localhost:%d/gw/backend/mtls-echo/hi", lib.GetPort()))
	if err != nil {
		t.Fatalf("Unexpected error when sending plain HTTP request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request when sending plain HTTP to a TLS-only port, got %d", resp.StatusCode)
	}
}
