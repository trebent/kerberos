package admin

import (
	"fmt"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	api "github.com/trebent/kerberos/internal/api/auth/admin"
	apierror "github.com/trebent/kerberos/internal/api/error"
	adminapi "github.com/trebent/kerberos/internal/auth/admin/api"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/oas"
	"github.com/trebent/zerologr"
)

type (
	Opts struct {
		Mux *http.ServeMux
		DB  db.SQLClient

		OASDir string

		ClientID     string
		ClientSecret string
	}
)

const (
	authAdminSpecification = "auth_admin.yaml"
	adminBasePath          = "/api/auth/admin"
)

func Init(opts *Opts) {
	zerologr.Info("Setting up administration API")

	data, err := os.ReadFile(fmt.Sprintf("%s/%s", opts.OASDir, authAdminSpecification))
	if err != nil {
		panic(fmt.Errorf("failed to read admin authentication OAS: %w", err))
	}

	spec, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		panic(fmt.Errorf("failed to load admin authentication OAS: %w", err))
	}

	ssi := adminapi.NewSSI(opts.DB, opts.ClientID, opts.ClientSecret)
	strictHandler := api.NewStrictHandlerWithOptions(
		ssi,
		[]api.StrictMiddlewareFunc{},
		api.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  apierror.RequestErrorHandler,
			ResponseErrorHandlerFunc: apierror.ResponseErrorHandler,
		},
	)

	_ = api.HandlerWithOptions(strictHandler, api.StdHTTPServerOptions{
		BaseURL:    adminBasePath,
		BaseRouter: opts.Mux,
		Middlewares: []api.MiddlewareFunc{
			oas.ValidationMiddleware(spec),
		},
	})
}
