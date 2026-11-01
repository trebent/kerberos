package composer_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/custom"
	"github.com/trebent/kerberos/internal/composer/debug"
	"github.com/trebent/kerberos/internal/composer/forwarder"
	obs "github.com/trebent/kerberos/internal/composer/observability"
	"github.com/trebent/kerberos/internal/composer/router"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/zerologr"
)

func BenchmarkComposer(b *testing.B) {
	cfg := config.NewKerberos()
	cfg.GatewayCfg.Router = &config.Router{
		Backends: []*config.RouterBackend{
			{Name: "backend", Host: "localhost", Port: 15000},
		},
	}
	cfg.PostProcess()

	zerologr.Set(logr.Discard())

	forwarder, err := forwarder.NewComponent(&forwarder.Opts{Backends: cfg.GatewayCfg.Router.Backends})
	if err != nil {
		b.Fatal(err.Error())
	}

	composer := composer.New(&composer.Opts{
		Observability: obs.NewComponent(&obs.Opts{Cfg: cfg.ObsCfg, Debugger: debug.NewDummy(nil)}),
		Router:        router.NewComponent(&router.Opts{Cfg: cfg.GatewayCfg.Router}),
		Custom:        custom.NewComponent(),
		Forwarder:     forwarder,
	})

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "http://localhost:15000/gw/backend/backend/hi", bytes.NewBuffer([]byte("hi")))

		composer.ServeHTTP(recorder, request)
	}
}
