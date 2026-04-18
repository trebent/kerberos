package oas

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/config"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
)

// TestComponentWithBodyValidation tests the OAS component with body validation enabled.
func TestComponentWithBodyValidation(t *testing.T) {
	mux := http.NewServeMux()
	opts := &Opts{
		Cfg: &config.OASConfig{
			Mappings: []*config.OASBackendMapping{
				{
					Backend:       "test-backend",
					Specification: "testspecs/test.yaml",
					Options: &config.OASBackendMappingOpts{
						ValidateBody: true,
					},
				},
			},
			Order: 1,
		},
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
	start.Next(c)

	// Fake final component to terminate the flow.
	c.Next(&composer.Dummy{
		CustomHandler: func(fc composer.FlowComponent, w http.ResponseWriter, r *http.Request) {},
	})

	// Use fake router as starting handler.
	mux.Handle("/", start)

	t.Log("Running request with empty body, fails body validation")
	executeRequest(t, &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{},
		Body:   http.NoBody,
	}, mux, http.StatusBadRequest)

	t.Log("Running request with invalid body, fails body validation")
	executeRequest(t, &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"not-exist": "value"}`))),
	}, mux, http.StatusBadRequest)

	t.Log("Running request with valid body, passes body validation")
	executeRequest(t, &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"uuid": "123"}`))),
	}, mux, http.StatusOK)
}

// TestComponentWithoutBodyValidation tests the OAS component with body validation disabled.
func TestComponentWithoutBodyValidation(t *testing.T) {
	mux := http.NewServeMux()
	opts := &Opts{
		Cfg: &config.OASConfig{
			Mappings: []*config.OASBackendMapping{
				{
					Backend:       "test-backend",
					Specification: "testspecs/test.yaml",
					Options: &config.OASBackendMappingOpts{
						ValidateBody: false,
					},
				},
			},
			Order: 1,
		},
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
	start.Next(c)

	// Fake final component to terminate the flow.
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

func TestComponentGetOAS(t *testing.T) {
	opts := &Opts{
		Cfg: &config.OASConfig{
			Mappings: []*config.OASBackendMapping{
				{
					Backend:       "test-backend",
					Specification: "testspecs/test.yaml",
					Options:       &config.OASBackendMappingOpts{},
				},
			},
			Order: 1,
		},
	}

	c := NewComponent(opts)

	data, err := c.GetOAS("test-backend")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedBytes, err := os.ReadFile("testspecs/test.yaml")
	if err != nil {
		t.Fatalf("Failed to read expected OAS file: %v", err)
	}

	if !bytes.Equal(data, expectedBytes) {
		t.Fatal("Expected OAS bytes do not match actual bytes")
	}

	_, err = c.GetOAS("non-existent-backend")
	if !errors.Is(err, apierror.ErrNotFound) {
		t.Fatal("Expected error for non-existent backend, got nil")
	}
}

func executeRequest(t *testing.T, req *http.Request, handler http.Handler, expectedStatus int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != expectedStatus {
		t.Fatalf("Expected status %d, got %d", expectedStatus, recorder.Code)
	}
}
