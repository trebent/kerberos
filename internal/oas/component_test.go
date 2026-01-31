package oas

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/trebent/kerberos/internal/composer/custom"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
	"github.com/trebent/kerberos/internal/config"
	"github.com/xeipuuv/gojsonschema"
)

func TestComponentWithOptions(t *testing.T) {
	// Test that the component can be created with a sample configuration.
	cfgMap := config.New(testConfigOpts())
	_, err := RegisterWith(cfgMap)
	if err != nil {
		t.Fatalf("Failed to register OAS config: %v", err)
	}

	cfgMap.MustLoad(configName, []byte(configExplicitOptions()))
	if err := cfgMap.Parse(); err != nil {
		t.Fatalf("Failed to parse OAS config: %v", err)
	}

	mux := http.NewServeMux()
	opts := &Opts{
		Cfg: cfgMap,
		Mux: mux,
	}

	start := &composertypes.Dummy{
		CustomHandler: func(fc composertypes.FlowComponent, w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, composertypes.BackendContextKey, "test-backend")
			fc.ServeHTTP(w, r.WithContext(ctx))
		},
	}
	c := New(opts)
	if c == nil {
		t.Fatal("Expected non-nil component")
	}
	start.Next(c)
	c.Next(&composertypes.Dummy{
		CustomHandler: func(fc composertypes.FlowComponent, w http.ResponseWriter, r *http.Request) {},
	})
	mux.Handle("/", start)

	recorder := httptest.NewRecorder()
	noBodyReq := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{},
		Body:   http.NoBody,
	}
	mux.ServeHTTP(recorder, noBodyReq)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	recorder = httptest.NewRecorder()
	invalidBodyReq := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"not-exist": "value"}`))),
	}
	mux.ServeHTTP(recorder, invalidBodyReq)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("Expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	recorder = httptest.NewRecorder()
	validBodyReq := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(`{"uuid": "123"}`))),
	}
	mux.ServeHTTP(recorder, validBodyReq)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, recorder.Code)
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

func testConfigOpts() *config.Opts {
	return &config.Opts{
		GlobalSchemas: []gojsonschema.JSONLoader{
			custom.OrderedSchemaJSONLoader(),
		},
	}
}
