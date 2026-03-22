package obs

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/response"
)

func TestObservabilityDisabled(t *testing.T) {
	cfg := &config.ObservabilityConfig{
		Enabled: false,
	}

	opts := &Opts{Cfg: cfg}
	component := NewComponent(opts)

	dummy := &composer.Dummy{
		CustomHandler: func(_ composer.FlowComponent, w http.ResponseWriter, req *http.Request) {
			_, err := logr.FromContext(req.Context())
			if err != nil {
				t.Fatal("logr.Logger should have been set")
			}

			_, ok := w.(*response.Wrapper)
			if !ok {
				t.Fatal("Expected ResponseWrapper, got something else")
			}

			_, ok = req.Body.(*response.BodyWrapper)
			if ok {
				t.Fatal("Unexpected BodyWrapper")
			}

			w.WriteHeader(http.StatusOK)
		},
	}

	component.Next(dummy)
	recorder := httptest.NewRecorder()
	component.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/gw/backend/one/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestObservability(t *testing.T) {
	cfg := &config.ObservabilityConfig{
		Enabled:        true,
		RuntimeMetrics: false,
	}

	opts := &Opts{Cfg: cfg}
	component := NewComponent(opts)

	dummy := &composer.Dummy{
		CustomHandler: func(_ composer.FlowComponent, w http.ResponseWriter, req *http.Request) {
			_, err := logr.FromContext(req.Context())
			if err != nil {
				t.Fatal("logr.Logger should have been set")
			}

			_, ok := w.(*response.Wrapper)
			if !ok {
				t.Fatal("Expected ResponseWrapper, got something else")
			}

			_, ok = req.Body.(*response.BodyWrapper)
			if !ok {
				t.Fatal("Expected BodyWrapper, got something else")
			}

			w.WriteHeader(http.StatusOK)
		},
	}

	component.Next(dummy)
	recorder := httptest.NewRecorder()
	component.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/gw/backend/one/", bytes.NewReader([]byte("hi"))))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
}
