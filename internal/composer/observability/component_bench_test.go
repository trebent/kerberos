package obs_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/debug"
	obs "github.com/trebent/kerberos/internal/composer/observability"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/zerologr"
)

func BenchmarkObs_Standard(b *testing.B) {
	zerologr.Set(logr.Discard())

	comp := obs.NewComponent(&obs.Opts{
		Cfg:      &config.Observability{Enabled: true, RuntimeMetrics: true},
		Version:  "1.0.0",
		Debugger: debug.NewDummy(nil),
	})
	dummy := composer.Dummy{
		CustomHandler: func(_ composer.FlowComponent, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	}
	comp.Next(&dummy)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "http://example.com/gw/backend/test/", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		comp.ServeHTTP(recorder, request)
	}
}
