package admin

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	adminext "github.com/trebent/kerberos/internal/admin/extensions"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/db"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/kerberos/internal/oas"
	"github.com/trebent/zerologr"
)

type (
	Opts struct {
		Mux *http.ServeMux
		DB  db.SQLClient

		OASDir string

		Cfg *config.AdminConfig
	}
	Admin struct {
		SessionMiddleware adminapi.StrictMiddlewareFunc

		ssi withExtensions
	}
)

//go:embed db/schema.sql
var dbschemaBytes []byte

const (
	authAdminSpecification = "admin.yaml"
	schemaApplyTimeout     = 10 * time.Second
)

// Runs the administration API.
func New(opts *Opts) *Admin {
	zerologr.Info("Setting up administration API")
	applySchemas(opts.DB)

	data, err := os.ReadFile(fmt.Sprintf("%s/%s", opts.OASDir, authAdminSpecification))
	if err != nil {
		panic(fmt.Errorf("failed to read admin authentication OAS: %w", err))
	}

	spec, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		panic(fmt.Errorf("failed to load admin authentication OAS: %w", err))
	}

	ssi := newSSI(&ssiOpts{
		DB:           opts.DB,
		ClientID:     opts.Cfg.SuperUser.ClientID,
		ClientSecret: opts.Cfg.SuperUser.ClientSecret,
	})

	adminSessionMiddleware := SessionMiddleware(ssi)
	strictHandler := adminapi.NewStrictHandlerWithOptions(
		ssi,
		[]adminapi.StrictMiddlewareFunc{
			RequireSessionMiddleware(),
			adminSessionMiddleware,
		},
		adminapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  apierror.RequestErrorHandler,
			ResponseErrorHandlerFunc: apierror.ResponseErrorHandler,
		},
	)

	_ = adminapi.HandlerWithOptions(strictHandler, adminapi.StdHTTPServerOptions{
		BaseRouter: opts.Mux,
		Middlewares: []adminapi.MiddlewareFunc{
			oas.ValidationMiddleware(spec),
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					zerologr.Info(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
					next.ServeHTTP(w, r)
				})
			},
		},
	})

	return &Admin{
		SessionMiddleware: adminSessionMiddleware,
		ssi:               ssi,
	}
}

func (a *Admin) SetFlowFetcher(ff adminext.FlowFetcher) {
	a.ssi.SetFlowFetcher(ff)
}

func (a *Admin) SetOASBackend(backend adminext.OASBackend) {
	a.ssi.SetOASBackend(backend)
}

func applySchemas(db db.SQLClient) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), schemaApplyTimeout)
	defer cancel()
	if _, err := db.Exec(timeoutCtx, string(dbschemaBytes)); err != nil {
		panic(err)
	}
}
