package basic

import (
	"net/http"

	"github.com/trebent/kerberos/internal/apierror"
	"github.com/trebent/kerberos/internal/auth/method"
	"github.com/trebent/kerberos/internal/auth/method/basic/api"
	"github.com/trebent/kerberos/internal/db"
)

type (
	basic struct {
		db db.SQLClient
	}
	Opts struct {
		Mux *http.ServeMux
		DB  db.SQLClient
	}
)

const basicBasePath = "/api/auth/basic"

var _ method.Method = (*basic)(nil)

// New will return an authentication method and register API endpoints with the input serve mux.
func New(opts *Opts) method.Method {
	b := &basic{
		db: opts.DB,
	}

	b.registerAPI(opts.Mux)
	return b
}

func (a *basic) Authenticated(*http.Request) error {
	// Read session info from the DB and compare it to the incoming request.
	return nil
}

func (a *basic) Authorized(*http.Request) error {
	return nil
}

func (a *basic) registerAPI(mux *http.ServeMux) {
	ssi := api.NewSSI(a.db)
	_ = api.HandlerFromMuxWithBaseURL(
		api.NewStrictHandlerWithOptions(
			ssi,
			[]api.StrictMiddlewareFunc{api.AuthMiddleware(ssi)},
			api.StrictHTTPServerOptions{
				RequestErrorHandlerFunc:  apierror.RequestErrorHandler,
				ResponseErrorHandlerFunc: apierror.ResponseErrorHandler,
			},
		),
		mux,
		basicBasePath,
	)
}
