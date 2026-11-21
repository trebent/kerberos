package security

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestSelectCORSMiddleware(t *testing.T) {
	t.Run("allowAll is true", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		handler := SelectCORSMiddleware(nil, true, false)(nextHandler)

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Result().StatusCode)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
			t.Errorf("Expected Access-Control-Allow-Origin header to be set to %s, got %s", "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("allowedOrigins is non-empty", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		allowedOrigins := []string{"http://allowed.com"}
		handler := SelectCORSMiddleware(allowedOrigins, false, false)(nextHandler)

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", "http://allowed.com")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Result().StatusCode)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "http://allowed.com" {
			t.Errorf("Expected Access-Control-Allow-Origin header to be set to %s, got %s", "http://allowed.com", w.Header().Get("Access-Control-Allow-Origin"))
		}

		req, _ = http.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", "http://not-allowed.com")
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusForbidden {
			t.Errorf("Expected status code %d for non-allowed origin, got %d", http.StatusForbidden, w.Result().StatusCode)
		}
	})

	t.Run("neither allowAll nor allowedOrigins is set", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		handler := SelectCORSMiddleware(nil, false, false)(nextHandler)

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Result().StatusCode)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("Expected Access-Control-Allow-Origin header to be empty, got %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("denyAll is set", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		handler := SelectCORSMiddleware(nil, false, true)(nextHandler)

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusForbidden {
			t.Errorf("Expected status code %d, got %d", http.StatusForbidden, w.Result().StatusCode)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("Expected Access-Control-Allow-Origin header to be empty, got %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}

func TestCORSVaryOrigin(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		allowAll       bool
		denyAll        bool
		origin         string
		wantVary       bool
	}{
		{
			name:     "allowAll with origin",
			allowAll: true,
			origin:   "http://example.com",
			wantVary: true,
		},
		{name: "allowAll without origin", allowAll: true, wantVary: false},
		{
			name:           "allowed origin",
			allowedOrigins: []string{"http://allowed.com"},
			origin:         "http://allowed.com",
			wantVary:       true,
		},
		{
			name:           "origin not allowed",
			allowedOrigins: []string{"http://allowed.com"},
			origin:         "http://not-allowed.com",
			wantVary:       true,
		},
		{
			name:           "allowedOrigins without origin",
			allowedOrigins: []string{"http://allowed.com"},
			wantVary:       false,
		},
		{name: "denyAll with origin", denyAll: true, origin: "http://example.com", wantVary: true},
		{name: "denyAll without origin", denyAll: true, wantVary: false},
		{name: "no CORS configuration", origin: "http://example.com", wantVary: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Add("Vary", "Accept-Encoding")
				w.WriteHeader(http.StatusOK)
			})
			handler := SelectCORSMiddleware(tt.allowedOrigins, tt.allowAll, tt.denyAll)(next)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			vary := w.Header().Values("Vary")
			if got := slices.Contains(vary, "Origin"); got != tt.wantVary {
				t.Fatalf("expected Vary: Origin to be %t, got Vary values %v", tt.wantVary, vary)
			}

			if w.Code == http.StatusOK && !slices.Contains(vary, "Accept-Encoding") {
				t.Fatalf("expected the handler's own Vary value to be kept, got %v", vary)
			}
		})
	}
}
