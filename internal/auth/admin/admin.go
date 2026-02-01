package admin

import (
	"net/http"

	api "github.com/trebent/kerberos/internal/api/auth/admin"
	apierror "github.com/trebent/kerberos/internal/api/error"
	adminapi "github.com/trebent/kerberos/internal/auth/admin/api"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"
)

type (
	Opts struct {
		Mux *http.ServeMux
		DB  db.SQLClient

		ClientID     string
		ClientSecret string
	}
)

const adminBasePath = "/api/auth/admin"

func Init(opts *Opts) {
	zerologr.Info("Setting up administration API")

	ssi := adminapi.NewSSI(opts.DB, opts.ClientID, opts.ClientSecret)
	_ = api.HandlerFromMuxWithBaseURL(
		api.NewStrictHandlerWithOptions(
			ssi,
			[]api.StrictMiddlewareFunc{},
			api.StrictHTTPServerOptions{
				RequestErrorHandlerFunc:  apierror.RequestErrorHandler,
				ResponseErrorHandlerFunc: apierror.ResponseErrorHandler,
			},
		),
		opts.Mux,
		adminBasePath,
	)
}
