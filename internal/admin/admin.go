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
		// Mux is the HTTP ServeMux on which the admin API is registered.
		mux *http.ServeMux

		// StrictServerInterface of the admin API. Used to manufacture middleware to late-registered
		// API providers.
		ssi withExtensions
	}
)

//go:embed dbschema/schema.sql
var dbschemaBytes []byte

const adminSpecification = "admin.yaml"

// Runs the administration API.
func New(opts *Opts) (*Admin, error) {
	zerologr.Info("Setting up administration API")

	if err := applySchemas(opts.SQLClient); err != nil {
		return nil, fmt.Errorf("failed to apply admin DB schema: %w", err)
	}

	data, err := os.ReadFile(fmt.Sprintf("%s/%s", opts.OASDir, adminSpecification))
	if err != nil {
		return nil, fmt.Errorf("failed to read admin OAS: %w", err)
	}

	spec, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load admin OAS: %w", err)
	}

	ssi, err := newSSI(&ssiOpts{
		SQLClient:    opts.SQLClient,
		ClientID:     opts.Cfg.SuperUser.ClientID,
		ClientSecret: opts.Cfg.SuperUser.ClientSecret,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SSI: %w", err)
	}

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
	}, nil
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

func applySchemas(sqlClient db.SQLClient) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), db.SchemaApplyTimeout)
	defer cancel()
	if _, err := sqlClient.Exec(timeoutCtx, string(dbschemaBytes)); err != nil {
		return err
	}
	return nil
}
