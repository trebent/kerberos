package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/trebent/kerberos/internal/db"
)

type failingSQLClient struct{ db.SQLClient }

func (failingSQLClient) Query(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("database unavailable")
}

func TestSessionLookupFailureReturnsISE(t *testing.T) {
	h, err := newHandler(opts{target: "127.0.0.1:1", scheme: "http", sqlClient: failingSQLClient{}})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Cookie", "session=a")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestMissingSessionCookieReturnsUnauthorized(t *testing.T) {
	h, err := newHandler(opts{target: "127.0.0.1:1", scheme: "http", sqlClient: failingSQLClient{}})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestProxyStripsKerberosCookies(t *testing.T) {
	var received string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get("Cookie")
	}))
	defer target.Close()

	targetURL, _ := url.Parse(target.URL)
	h, err := newHandler(opts{target: targetURL.Host, scheme: targetURL.Scheme})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Cookie", "session=a; refresh=b; csrf=c; theme=dark")

	h.proxy.ServeHTTP(httptest.NewRecorder(), req)

	if received != "theme=dark" {
		t.Errorf("expected target to receive Cookie %q, got %q", "theme=dark", received)
	}
}
