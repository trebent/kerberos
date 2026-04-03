package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/response"
)

func TestRouter(t *testing.T) {
	cfg := &config.RouterConfig{
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
	router.ServeHTTP(wrapped, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, recorder.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/gw/backend/backend1/some/path", nil)
	recorder = httptest.NewRecorder()
	wrapped = response.NewResponseWrapper(recorder)
	router.ServeHTTP(wrapped, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status code %d, got %d", http.StatusNoContent, recorder.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/gw/backenddddd/backend1/some/path", nil)
	recorder = httptest.NewRecorder()
	wrapped = response.NewResponseWrapper(recorder)
	router.ServeHTTP(wrapped, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}
