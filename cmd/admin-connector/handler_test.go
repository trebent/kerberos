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
