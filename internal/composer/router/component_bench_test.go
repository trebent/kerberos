package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/router"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/response"
)

func BenchmarkRouter_SingleBackend(b *testing.B) {
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
	dummy := &composer.Dummy{
		CustomHandler: func(_ composer.FlowComponent, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	}
	comp.Next(dummy)

	req := httptest.NewRequest("GET", "/gw/backend/backend1/test", nil)

	b.ResetTimer()
	for b.Loop() {
		recorder := httptest.NewRecorder()
		wrapper := response.NewResponseWrapper(recorder)
		comp.ServeHTTP(wrapper, req.Clone(b.Context()))

		if recorder.Code != http.StatusOK {
			b.Log("Response body:", recorder.Body.String())
			b.Fatalf("expected status code 200, got %d", recorder.Code)
		}
	}
}
