package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProxyStripsKerberosCookies(t *testing.T) {
	var received string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	targetURL, err := url.Parse(target.URL)
	if err != nil {
		t.Fatalf("Failed to parse target URL: %v", err)
	}

	h, err := newHandler(opts{target: targetURL.Host, scheme: targetURL.Scheme})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboards", nil)
	req.Header.Set("Cookie", "session=abc; grafana_session=xyz; refresh=def; csrf=ghi")

	recorder := httptest.NewRecorder()
	h.proxy.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
	if received != "grafana_session=xyz" {
		t.Errorf("Expected target to receive Cookie %q, got %q", "grafana_session=xyz", received)
	}
}
