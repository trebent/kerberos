package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/response"
)

func TestRouter(t *testing.T) {
	cfg := &config.Router{
		Backends: []*config.RouterBackend{
			{
				Name: "backend1",
				Host: "localhost",
				Port: 8080,
			},
			{
				Name: "backend2",
				Host: "localhost",
				Port: 8081,
			},
		},
	}

	dummy := composer.Dummy{
		CustomHandler: func(_ composer.FlowComponent, w http.ResponseWriter, req *http.Request) {
			t.Log("Reached dummy handler")
			if req.URL.Path != "/some/path" {
				t.Fatalf("Expected URL to have a stripped path, got %s", req.URL.Path)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	}

	opts := &Opts{Cfg: cfg}
	router := NewComponent(opts)
	router.Next(&dummy)

	recorder := httptest.NewRecorder()
	wrapped := response.NewResponseWrapper(recorder)
	req := httptest.NewRequest(http.MethodGet, "/gw/backend/not-exist/", nil)
	router.ServeHTTP(wrapped, req.WithContext(context.WithValue(req.Context(), composer.BackendContextKey, "not-exist")))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, recorder.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/gw/backend/backend1/some/path", nil)
	recorder = httptest.NewRecorder()
	wrapped = response.NewResponseWrapper(recorder)
	router.ServeHTTP(wrapped, req.WithContext(context.WithValue(req.Context(), composer.BackendContextKey, cfg.Backends[0].Name)))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status code %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestRouterVaryOrigin(t *testing.T) {
	cfg := &config.Router{
		Backends: []*config.RouterBackend{
			{
				Name:    "cors",
				Host:    "localhost",
				Port:    8080,
				Origins: &config.Origins{AllowedOrigins: []string{"https://app.example.com"}},
			},
			{
				Name: "plain",
				Host: "localhost",
				Port: 8081,
			},
		},
	}

	tests := []struct {
		name       string
		backend    string
		origin     string
		wantStatus int
		wantVary   bool
	}{
		{
			name:       "allowed origin",
			backend:    "cors",
			origin:     "https://app.example.com",
			wantStatus: http.StatusNoContent,
			wantVary:   true,
		},
		{
			name:       "origin not allowed",
			backend:    "cors",
			origin:     "https://evil.example.com",
			wantStatus: http.StatusForbidden,
			wantVary:   true,
		},
		{name: "no origin", backend: "cors", wantStatus: http.StatusNoContent, wantVary: false},
		{
			name:       "backend without CORS",
			backend:    "plain",
			origin:     "https://app.example.com",
			wantStatus: http.StatusNoContent,
			wantVary:   false,
		},
	}

	router := NewComponent(&Opts{Cfg: cfg})
	router.Next(&composer.Dummy{
		CustomHandler: func(_ composer.FlowComponent, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/gw/backend/"+tt.backend+"/path", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			recorder := httptest.NewRecorder()
			ctx := context.WithValue(req.Context(), composer.BackendContextKey, tt.backend)
			router.ServeHTTP(response.NewResponseWrapper(recorder), req.WithContext(ctx))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status code %d, got %d", tt.wantStatus, recorder.Code)
			}

			vary := recorder.Header().Values("Vary")
			if got := slices.Contains(vary, "Origin"); got != tt.wantVary {
				t.Fatalf("expected Vary: Origin to be %t, got Vary values %v", tt.wantVary, vary)
			}
		})
	}
}
