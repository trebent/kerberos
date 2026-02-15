package admin

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	adminapi "github.com/trebent/kerberos/internal/admin/api"
	adminapigen "github.com/trebent/kerberos/internal/api/admin"
	apierror "github.com/trebent/kerberos/internal/api/error"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/oas"
	"github.com/trebent/zerologr"
)

type (
	Opts struct {
		Mux *http.ServeMux
		DB  db.SQLClient

		OASDir string

		Cfg config.Map
	}
)

//go:embed dbschema/schema.sql
var dbschemaBytes []byte

const (
	authAdminSpecification = "admin.yaml"
	schemaApplyTimeout     = 10 * time.Second
)

// Runs the administration API.
func Init(opts *Opts) {
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

	cfg := config.AccessAs[*adminConfig](opts.Cfg, configName)
	ssi := adminapi.NewSSI(opts.DB, cfg.SuperUser.ClientID, cfg.SuperUser.ClientSecret)

	strictHandler := adminapigen.NewStrictHandlerWithOptions(
		ssi,
		[]adminapigen.StrictMiddlewareFunc{},
		adminapigen.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  apierror.RequestErrorHandler,
			ResponseErrorHandlerFunc: apierror.ResponseErrorHandler,
		},
	)

	_ = adminapigen.HandlerWithOptions(strictHandler, adminapigen.StdHTTPServerOptions{
		BaseRouter: opts.Mux,
		Middlewares: []adminapigen.MiddlewareFunc{
			oas.ValidationMiddleware(spec),
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					zerologr.Info(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
					next.ServeHTTP(w, r)
				})
			},
		},
	})
}

func applySchemas(db db.SQLClient) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), schemaApplyTimeout)
	defer cancel()
	if _, err := db.Exec(timeoutCtx, string(dbschemaBytes)); err != nil {
		panic(err)
	}
}
