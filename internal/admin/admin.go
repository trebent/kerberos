package admin

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"

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
		// Mux is the HTTP ServeMux on which the admin API will be registered.
		Mux *http.ServeMux

		// SQLClient for the admin API to use.
		SQLClient db.SQLClient

		// Directory where OAS for the admin API can be found.
		OASDir string

		// Admin configuration.
		Cfg *config.AdminConfig
	}
	Admin struct {
		mux *http.ServeMux
		ssi withExtensions
	}
)

//go:embed db/schema.sql
var dbschemaBytes []byte

const authAdminSpecification = "admin.yaml"

// Runs the administration API.
func New(opts *Opts) *Admin {
	zerologr.Info("Setting up administration API")
	applySchemas(opts.SQLClient)

	data, err := os.ReadFile(fmt.Sprintf("%s/%s", opts.OASDir, authAdminSpecification))
	if err != nil {
		panic(fmt.Errorf("failed to read admin authentication OAS: %w", err))
	}

	spec, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		panic(fmt.Errorf("failed to load admin authentication OAS: %w", err))
	}

	ssi := newSSI(&ssiOpts{
		SQLClient:    opts.SQLClient,
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
		ssi: ssi,
		mux: opts.Mux,
	}
}

// SetFlowFetcher sets the flow fetcher for the admin component. This allows the admin API to serve flow metadata
// API calls.
func (a *Admin) SetFlowFetcher(ff adminext.FlowFetcher) {
	a.ssi.SetFlowFetcher(ff)
}

// SetOASBackend sets the OAS backend for the admin component. This allows the admin API to serve OAS
// to clients per backend providing an OAS.
func (a *Admin) SetOASBackend(backend adminext.OASBackend) {
	a.ssi.SetOASBackend(backend)
}

// RegisterAPIProvider registers an API provider with the admin API. All adminext.APIProvider implementations must
// be registered using this method in order for their routes to be served by the admin API.
func (a *Admin) RegisterAPIProvider(apiProvider adminext.APIProvider) error {
	return apiProvider.RegisterRoutes(
		a.mux,
		SessionMiddleware(a.ssi),
	)
}

func applySchemas(sqlClient db.SQLClient) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), db.SchemaApplyTimeout)
	defer cancel()
	if _, err := sqlClient.Exec(timeoutCtx, string(dbschemaBytes)); err != nil {
		panic(err)
	}
}
