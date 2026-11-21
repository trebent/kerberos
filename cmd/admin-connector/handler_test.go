package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	admindb "github.com/trebent/kerberos/internal/admin/db"
	"github.com/trebent/kerberos/internal/db/sqlite"
)

// TestConnectorStripsKerberosCookies sends a real HTTP request with a valid admin session through
// the connector, served by an httptest server, and checks the Cookie header that reaches the target.
func TestConnectorStripsKerberosCookies(t *testing.T) {
	sqlClient := sqlite.New(&sqlite.Opts{DSN: filepath.Join(t.TempDir(), "connector.db")})
	if err := admindb.ApplySchemas(sqlClient); err != nil {
		t.Fatalf("Failed to apply admin schemas: %v", err)
	}
	if err := admindb.BootstrapSuperuser(sqlClient, "admin", "secret"); err != nil {
		t.Fatalf("Failed to bootstrap superuser: %v", err)
	}
	superuser, err := admindb.GetSuperuser(t.Context(), sqlClient)
	if err != nil {
		t.Fatalf("Failed to get superuser: %v", err)
	}
	const sessionID = "connector-test-session"
	if err := admindb.CreateSession(
		t.Context(), sqlClient, superuser.ID, "connector-test-refresh", sessionID,
	); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	var (
		reached  bool
		received []string
	)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		received = r.Header.Values("Cookie")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(target.Close)

	targetURL, err := url.Parse(target.URL)
	if err != nil {
		t.Fatalf("Failed to parse target URL: %v", err)
	}

	h, err := newHandler(opts{target: targetURL.Host, scheme: targetURL.Scheme, sqlClient: sqlClient})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	connector := httptest.NewServer(h)
	t.Cleanup(connector.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, connector.URL+"/dashboards", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(
		"Cookie",
		"session="+sessionID+"; grafana_session=xyz; refresh=connector-test-refresh; csrf=t0k3n",
	)

	resp, err := connector.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if !reached {
		t.Fatal("Request did not reach the target")
	}
	if len(received) != 1 || received[0] != "grafana_session=xyz" {
		t.Errorf("Expected target to receive Cookie %q, got %q", "grafana_session=xyz", received)
	}
}
