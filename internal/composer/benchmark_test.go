package composer_test

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = io.Copy(io.Discard, req.Body)
		w.WriteHeader(http.StatusOK)
	}))
	b.Cleanup(srv.Close)
	addr, _ := srv.Listener.Addr().(*net.TCPAddr)

	cfg := config.NewKerberos()
	cfg.GatewayCfg.Router = &config.Router{
		Backends: []*config.RouterBackend{
			{Name: "backend", Host: addr.IP.String(), Port: addr.Port},
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

	path := "/gw/backend/backend/hi"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, bytes.NewBuffer([]byte("hi")))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		clear(recorder.Header())

		request.URL.Path = path
		request.Body = io.NopCloser(strings.NewReader("hi"))
		composer.ServeHTTP(recorder, request)
	}

	if recorder.Result().StatusCode != http.StatusOK {
		b.Fatal("Expected HTTP status 200")
	}
}
