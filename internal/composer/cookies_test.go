package composer_test

import (
	"net"
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

// TestGatewayStripsKerberosCookies sends real HTTP requests through the full gateway pipeline,
// served by an httptest server, and checks the Cookie header that reaches the backend.
func TestGatewayStripsKerberosCookies(t *testing.T) {
	type received struct {
		cookie  []string
		reached bool
	}
	got := make(chan received, 1)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- received{cookie: r.Header.Values("Cookie"), reached: true}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(backend.Close)
	addr, _ := backend.Listener.Addr().(*net.TCPAddr)

	cfg := config.NewKerberos()
	cfg.GatewayCfg.Router = &config.Router{
		Backends: []*config.RouterBackend{
			{Name: "backend", Host: addr.IP.String(), Port: addr.Port},
		},
	}
	cfg.PostProcess()

	zerologr.Set(logr.Discard())

	fwd, err := forwarder.NewComponent(&forwarder.Opts{Backends: cfg.GatewayCfg.Router.Backends})
	if err != nil {
		t.Fatal(err)
	}

	gateway := httptest.NewServer(composer.New(&composer.Opts{
		Observability: obs.NewComponent(&obs.Opts{Cfg: cfg.ObsCfg, Debugger: debug.NewDummy(nil)}),
		Router:        router.NewComponent(&router.Opts{Cfg: cfg.GatewayCfg.Router}),
		Custom:        custom.NewComponent(),
		Forwarder:     fwd,
	}))
	t.Cleanup(gateway.Close)

	tests := []struct {
		name   string
		cookie string
		want   []string
	}{
		{
			name:   "Kerberos cookies removed, others kept",
			cookie: "session=s3cr3t; theme=dark; refresh=r3fr3sh; csrf=t0k3n; lang=en",
			want:   []string{"theme=dark; lang=en"},
		},
		{
			name:   "only Kerberos cookies, no Cookie header reaches the backend",
			cookie: "session=s3cr3t; refresh=r3fr3sh; csrf=t0k3n",
			want:   nil,
		},
		{
			name:   "no Kerberos cookies, header unchanged",
			cookie: "theme=dark",
			want:   []string{"theme=dark"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(
				t.Context(), http.MethodGet, gateway.URL+"/gw/backend/backend/hello", nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Cookie", tt.cookie)

			resp, err := gateway.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
			}

			r := <-got
			if !r.reached {
				t.Fatal("request did not reach the backend")
			}
			if len(r.cookie) != len(tt.want) || (len(tt.want) > 0 && r.cookie[0] != tt.want[0]) {
				t.Fatalf("expected backend to receive Cookie %q, got %q", tt.want, r.cookie)
			}
		})
	}
}
