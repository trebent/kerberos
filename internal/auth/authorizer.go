package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	_ "embed"

	"github.com/trebent/kerberos/internal/apierror"
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

	errNoMethod           = errors.New("no authentication method defined")
	errUnrecognizedMethod = errors.New("unrecognized authentication method")
	errAuth               = errors.New("you do not have permission to do that")
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
	ctx := req.Context()
	//nolint:errcheck // if this isn't populated the flow chain has been broken.
	backend := ctx.Value(composertypes.BackendContextKey).(string)

	m, err := a.findMethod(backend)
	if err != nil && errors.Is(err, errNoMethod) {
		zerologr.V(20).
			Info(fmt.Sprintf("Backend %s does not have a defined auth method, calling next", backend))
		a.next.ServeHTTP(w, req)
		return
	} else if err != nil {
		zerologr.Error(err, "Error during authentication")
		response.JSONError(w, apierror.ErrInternal, http.StatusInternalServerError)
		return
	}

	if err := m.Authenticated(req); err != nil {
		zerologr.Error(err, "User tried to perform an authenticated action while unauthenticated")
		response.JSONError(w, errAuth, http.StatusUnauthorized)
		return
	}

	if err := m.Authorized(req); err != nil {
		zerologr.Error(err, "User tried to perform an action they were not authorized to do")
		response.JSONError(w, errAuth, http.StatusForbidden)
		return
	}

	// Forward the request now that it's been auth'd.
	a.next.ServeHTTP(w, req)
}

// findMethod attempts to find the method which protects the input backend, if any.
func (a *authorizer) findMethod(backend string) (method.Method, error) {
	for _, mapping := range a.cfg.Scheme.Mappings {
		if mapping.Backend == backend {
			switch mapping.Method {
			case "basic":
				zerologr.V(50).Info("Using basic authentication for backend: " + backend)
				return a.basic, nil
			default:
				return nil, errUnrecognizedMethod
			}
		}
	}

	return nil, errNoMethod
}

func (a *authorizer) applySchemas() {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), schemaApplyTimeout)
	defer cancel()
	if _, err := a.db.Exec(timeoutCtx, string(dbschemaBytes)); err != nil {
		panic(err)
	}
}
