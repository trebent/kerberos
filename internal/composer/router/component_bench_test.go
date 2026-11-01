package router_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/router"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/zerologr"
)

func BenchmarkRouter_SingleBackend(b *testing.B) {
	zerologr.Set(logr.Discard())

	comp := router.NewComponent(&router.Opts{
		Cfg: &config.Router{
			Backends: []*config.RouterBackend{
				{
					Name: "backend1",
					Host: "localhost",
					Port: 8080,
				},
			},
		},
	})
	comp.Next(&composer.Noop{})

	path := "/gw/backend/backend1/test"

	recorder := httptest.NewRecorder()
	wrapper := response.NewResponseWrapper(recorder)
	request := httptest.NewRequest(http.MethodGet, path, bytes.NewBuffer([]byte{}))
	request = request.WithContext(context.WithValue(request.Context(), composer.BackendContextKey, "backend1"))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		request.URL.Path = path
		comp.ServeHTTP(wrapper, request)

		if recorder.Code != http.StatusOK {
			b.Log("Response body:", recorder.Body.String())
			b.Fatalf("expected status code 200, got %d", recorder.Code)
		}
	}
}
