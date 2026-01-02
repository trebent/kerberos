package auth

import (
	"errors"
	"net/http"

	"github.com/trebent/kerberos/internal/auth/method"
	"github.com/trebent/kerberos/internal/auth/method/basic"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/zerologr"
)

type (
	Opts struct {
		Cfg config.Map

		// The Mux to register the basic authentication API with, if enabled.
		Mux *http.ServeMux
	}
	authorizer struct {
		next composertypes.FlowComponent

		cfg   *authConfig
		basic method.Method
	}
)

var (
	_ composertypes.FlowComponent = (*authorizer)(nil)

	errAuth = errors.New("you do not have permission to do that")
)

func New(opts *Opts) composertypes.FlowComponent {
	cfg := config.AccessAs[*authConfig](opts.Cfg, configName)
	authorizer := &authorizer{cfg: cfg}

	if cfg.BasicEnabled() {
		// If basic auth, populate the mux with http endpoints.
		authorizer.registerBasicAPI(opts)
	}

	return authorizer
}

func (a *authorizer) Next(next composertypes.FlowComponent) {
	a.next = next
}

func (a *authorizer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if err := a.authenticated(req); err != nil {
		zerologr.Error(err, "User tried to perform an authenticated action while unauthenticated")
		response.JSONError(w, errAuth, http.StatusUnauthorized)
		return
	}

	if err := a.authorized(req); err != nil {
		zerologr.Error(err, "User tried to perform an action they were not authorized to do")
		response.JSONError(w, errAuth, http.StatusForbidden)
		return
	}

	// Forward the request now that it's been auth'd.
	a.next.ServeHTTP(w, req)
}

func (a *authorizer) authenticated(_ *http.Request) error {
	// TODO: check if the route is protected, check which method is used to protect the route, call the configured auth method.
	return nil
}

func (a *authorizer) authorized(_ *http.Request) error {
	// TODO: check if the route is protected, check which method is used to protect the route, call the configured auth method.
	return nil
}

func (a *authorizer) registerBasicAPI(opts *Opts) {
	a.basic = basic.New(opts.Mux)
}
