package obs_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/trebent/kerberos/internal/composer"
	obs "github.com/trebent/kerberos/internal/composer/observability"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/zerologr"
)

func BenchmarkObs_Standard(b *testing.B) {
	zerologr.Set(zerologr.New(&zerologr.Opts{Console: true, V: 20}))

	comp := obs.NewComponent(&obs.Opts{
		Cfg:     &config.ObservabilityConfig{},
		Version: "1.0.0",
	})
	dummy := composer.Dummy{
		CustomHandler: func(_ composer.FlowComponent, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	}
	comp.Next(&dummy)

	req := httptest.NewRequest("GET", "http://example.com/test", nil)

	b.ResetTimer()
	for b.Loop() {
		recorder := httptest.NewRecorder()
		comp.ServeHTTP(recorder, req)
	}
}
