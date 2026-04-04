package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"time"

	_ "embed"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/auth/method"
	"github.com/trebent/kerberos/internal/auth/method/basic"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/custom"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/db"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	authbasicapi "github.com/trebent/kerberos/internal/oapi/auth/basic"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/zerologr"
)

type (
	Opts struct {
		Cfg *config.AuthConfig

		// The Mux to register the basic authentication API with, if enabled.
		Mux *http.ServeMux

		DB db.SQLClient

		// Directory where OAS for the auth APIs can be found.
		OASDir string

		// To verify administrator callers, adds context into call flows to be able to determine if the caller is an admin user.
		AdminSessionMiddleware authbasicapi.StrictMiddlewareFunc
	}
	authorizer struct {
		next composer.FlowComponent

		cfg   *config.AuthConfig
		basic method.Method
		db    db.SQLClient
	}
)

var (
	_ composer.FlowComponent = (*authorizer)(nil)
	_ custom.Ordered         = (*authorizer)(nil)

	//go:embed dbschema/schema.sql
	dbschemaBytes []byte

	errExempted           = errors.New("path is exempted from auth")
	errNoMethod           = errors.New("no authentication method defined")
	errUnrecognizedMethod = errors.New("unrecognized authentication method")
)

const schemaApplyTimeout = 10 * time.Second

func NewComponent(opts *Opts) composer.FlowComponent {
	authorizer := &authorizer{
		cfg: opts.Cfg,
		db:  opts.DB,
	}
	authorizer.applySchemas()

	if opts.Cfg.Methods.Basic != nil {
		zerologr.Info("Basic authentication enabled")
		// If basic auth, create the method.
		authorizer.basic = basic.New(&basic.Opts{
			Mux:                    opts.Mux,
			DB:                     opts.DB,
			OASDir:                 opts.OASDir,
			AuthZConfig:            makeAuthZMap(opts.Cfg.Scheme.Mappings),
			AdminSessionMiddleware: opts.AdminSessionMiddleware,
		})
	}

	return authorizer
}

func (a *authorizer) Order() int {
	return a.cfg.Order
}

func (a *authorizer) Next(next composer.FlowComponent) {
	a.next = next
}

// GetMeta implements [composer.FlowComponent].
func (a *authorizer) GetMeta() []adminapi.FlowMeta {
	fmd := adminapi.FlowMeta_Data{}
	if err := fmd.FromFlowMetaDataAuth(adminapi.FlowMetaDataAuth{
		Methods: &adminapi.FlowMetaDataAuthMethods{
			Basic: func() *adminapi.FlowMetaDataAuthMethodBasic {
				if a.cfg.Methods.Basic == nil {
					return nil
				}
				return &adminapi.FlowMetaDataAuthMethodBasic{}
			}(),
			// Future auth methods would be added here.
		},
		Scheme: &adminapi.FlowMetaDataAuthScheme{
			Mappings: func() *[]adminapi.FlowMetaDataAuthSchemeMapping {
				var mappings []adminapi.FlowMetaDataAuthSchemeMapping
				for _, mapping := range a.cfg.Scheme.Mappings {
					mappings = append(mappings, adminapi.FlowMetaDataAuthSchemeMapping{
						Backend: mapping.Backend,
						Method:  mapping.Method,
						Exempt:  &mapping.Exempt,
						Authorization: func() *adminapi.FlowMetaDataAuthSchemeMappingAuthorization {
							if mapping.Authorization == nil {
								return nil
							}
							return &adminapi.FlowMetaDataAuthSchemeMappingAuthorization{
								Groups: &mapping.Authorization.Groups,
								Paths:  &mapping.Authorization.Paths,
							}
						}(),
					})
				}
				return &mappings
			}(),
		},
	}); err != nil {
		panic(err)
	}

	return append([]adminapi.FlowMeta{
		{
			Name: "authorizer",
			Data: fmd,
		},
	}, a.next.GetMeta()...)
}

func (a *authorizer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger, _ := logr.FromContext(ctx)
	logger = logger.WithName("authorizer")
	logger.Info("Authorizing request")
	//nolint:errcheck // if this isn't populated the flow chain has been broken.
	backend := ctx.Value(composer.BackendContextKey).(string)

	m, err := a.findMethod(backend, req)
	switch {
	case errors.Is(err, errNoMethod):
		zerologr.V(20).
			Info(fmt.Sprintf("Backend %s does not have a defined auth method, calling next", backend))
		a.next.ServeHTTP(w, req)
		return
	case errors.Is(err, errExempted):
		zerologr.V(20).
			Info(fmt.Sprintf("Backend %s path %s is exempted, calling next", backend, req.URL.Path))
		a.next.ServeHTTP(w, req)
		return
	case err != nil:
		zerologr.Error(err, "Error during authentication")
		apierror.ErrorHandler(w, req, apierror.APIErrInternal)
		return
	}

	if err := m.Authenticated(req); err != nil {
		zerologr.Error(err, "User tried to perform an authenticated action while unauthenticated")
		apierror.ErrorHandler(w, req, apierror.APIErrNoPermission)
		return
	}

	if err := m.Authorized(req); err != nil {
		zerologr.Error(err, "User tried to perform an action they were not authorized to do")
		apierror.ErrorHandler(w, req, apierror.APIErrForbidden)
		return
	}

	// Forward the request now that it's been auth'd.
	a.next.ServeHTTP(w, req)
}

// findMethod attempts to find the method which protects the input backend, if any.
func (a *authorizer) findMethod(backend string, req *http.Request) (method.Method, error) {
	for _, mapping := range a.cfg.Scheme.Mappings {
		if mapping.Backend == backend {
			switch mapping.Method {
			case "basic":
				zerologr.V(20).Info("Using basic authentication for backend: " + backend)
				for _, exemption := range mapping.Exempt {
					match, err := path.Match(exemption, req.URL.Path)
					if err != nil {
						return nil, err
					}

					if match {
						return nil, fmt.Errorf("%w: %s", errExempted, req.URL.Path)
					}
				}
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

func makeAuthZMap(mappings []*config.AuthMapping) map[string]*config.AuthZ {
	m := make(map[string]*config.AuthZ)
	for _, mapping := range mappings {
		m[mapping.Backend] = mapping.Authorization
	}
	return m
}
