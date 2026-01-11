package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	_ "embed"

	"github.com/trebent/kerberos/internal/auth/admin"
	"github.com/trebent/kerberos/internal/auth/method"
	"github.com/trebent/kerberos/internal/auth/method/basic"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/zerologr"
)

type (
	Opts struct {
		Cfg config.Map

		// The Mux to register the basic authentication API with, if enabled.
		Mux *http.ServeMux

		DB db.SQLClient
	}
	authorizer struct {
		next composertypes.FlowComponent

		cfg   *authConfig
		basic method.Method
		db    db.SQLClient
	}
)

var (
	_ composertypes.FlowComponent = (*authorizer)(nil)

	//go:embed dbschema/schema.sql
	dbschemaBytes []byte

	errAuth = errors.New("you do not have permission to do that")
)

const schemaApplyTimeout = 10 * time.Second

func New(opts *Opts) composertypes.FlowComponent {
	cfg := config.AccessAs[*authConfig](opts.Cfg, configName)
	authorizer := &authorizer{
		cfg: cfg,
		db:  opts.DB,
	}
	authorizer.applySchemas()

	if cfg.BasicEnabled() {
		zerologr.Info("Basic authentication enabled")
		// If basic auth, create the method.
		authorizer.basic = basic.New(&basic.Opts{
			Mux: opts.Mux,
			DB:  opts.DB,
		})
	}

	if cfg.AdministrationEnabled() {
		zerologr.Info("Administration enabled")
		admin.Init(&admin.Opts{
			Mux:          opts.Mux,
			DB:           opts.DB,
			ClientID:     cfg.Administration.SuperUser.ClientID,
			ClientSecret: cfg.Administration.SuperUser.ClientSecret,
		})
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

func (a *authorizer) applySchemas() {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), schemaApplyTimeout)
	defer cancel()
	if _, err := a.db.Exec(timeoutCtx, string(dbschemaBytes)); err != nil {
		panic(err)
	}
}
