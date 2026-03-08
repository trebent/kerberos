package oas

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/config"
)

// TestComponentWithOptions tests the OAS component with explicit options.
func TestComponentWithOptions(t *testing.T) {
	mux := http.NewServeMux()
	opts := &Opts{
		Cfg: &config.OASConfig{},
		Mux: mux,
	}

	// Fake router to populate the request context with backend name.
	start := &composer.Dummy{
		CustomHandler: func(fc composer.FlowComponent, w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, composer.BackendContextKey, "test-backend")
			fc.ServeHTTP(w, r.WithContext(ctx))
		},
	}
	c := NewComponent(opts)
	if c == nil {
		t.Fatal("Expected non-nil component")
	}
	start.Next(c)
	c.Next(&composer.Dummy{
		CustomHandler: func(fc composer.FlowComponent, w http.ResponseWriter, r *http.Request) {},
	})
	mux.Handle("/", start)

	executeRequest(t, &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{},
		Body:   http.NoBody,
	}, mux, http.StatusBadRequest)

	executeRequest(t, &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"not-exist": "value"}`))),
	}, mux, http.StatusBadRequest)

	executeRequest(t, &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"uuid": "123"}`))),
	}, mux, http.StatusOK)
}

// TestComponentWithoutOptions tests the OAS component with default options.
func TestComponentWithoutOptions(t *testing.T) {
	mux := http.NewServeMux()
	opts := &Opts{
		Cfg: &config.OASConfig{},
		Mux: mux,
	}

	start := &composer.Dummy{
		CustomHandler: func(fc composer.FlowComponent, w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, composer.BackendContextKey, "test-backend")
			fc.ServeHTTP(w, r.WithContext(ctx))
		},
	}
	c := NewComponent(opts)
	if c == nil {
		t.Fatal("Expected non-nil component")
	}
	start.Next(c)
	c.Next(&composer.Dummy{
		CustomHandler: func(fc composer.FlowComponent, w http.ResponseWriter, r *http.Request) {},
	})
	mux.Handle("/", start)

	executeRequest(t, &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"not-exist": "123"}`))),
	}, mux, http.StatusBadRequest)
}

// TestComponentWithoutBodyValidation tests the OAS component with body validation disabled.
func TestComponentWithoutBodyValidation(t *testing.T) {
	mux := http.NewServeMux()
	opts := &Opts{
		Cfg: &config.OASConfig{},
		Mux: mux,
	}

	start := &composer.Dummy{
		CustomHandler: func(fc composer.FlowComponent, w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, composer.BackendContextKey, "test-backend")
			fc.ServeHTTP(w, r.WithContext(ctx))
		},
	}
	c := NewComponent(opts)
	if c == nil {
		t.Fatal("Expected non-nil component")
	}
	start.Next(c)
	c.Next(&composer.Dummy{
		CustomHandler: func(fc composer.FlowComponent, w http.ResponseWriter, r *http.Request) {},
	})
	mux.Handle("/", start)

	executeRequest(t, &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"not-exist": "123"}`))),
	}, mux, http.StatusOK)
}

func executeRequest(t *testing.T, req *http.Request, handler http.Handler, expectedStatus int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != expectedStatus {
		t.Fatalf("Expected status %d, got %d", expectedStatus, recorder.Code)
	}
}

func configExplicitOptions() string {
	return `
{
	"mappings": [
		{
			"backend": "test-backend",
			"specification": "testspecs/test.yaml",
			"options": {
				"validateBody": true
			}
		}
	],
	"order": 1
}`
}

func configWithoutOptions() string {
	return `
{
	"mappings": [
		{
			"backend": "test-backend",
			"specification": "testspecs/test.yaml"
		}
	],
	"order": 1
}`
}

func configDisableBodyValidation() string {
	return `
{
	"mappings": [
		{
			"backend": "test-backend",
			"specification": "testspecs/test.yaml",
			"options": {
				"validateBody": false
			}
		}
	],
	"order": 1
}`
}
