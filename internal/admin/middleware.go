package admin

import (
	"context"
	"net/http"
	"time"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/trebent/kerberos/internal/admin/model"
	adminapigen "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/zerologr"
)

type adminContextKey int

const (
	adminContextIsSuperUser adminContextKey = 0
	adminContextSession     adminContextKey = 1
)

// SessionMiddleware provides context population of administration session information.
// If put in front of a handler, the context to the handler will contain information that can
// be used to determine if a KRB admin made the call, or if it was external.
func SessionMiddleware(
	ssi adminapigen.StrictServerInterface,
) adminapigen.StrictMiddlewareFunc {
	apiImpl, ok := ssi.(*impl)
	if !ok {
		panic("expected admin api *impl")
	}

	return func(
		f nethttp.StrictHTTPHandlerFunc,
		_ string,
	) nethttp.StrictHTTPHandlerFunc {
		return func(
			ctx context.Context,
			w http.ResponseWriter,
			r *http.Request,
			request any,
		) (any, error) {
			zerologr.V(20).Info("Running admin session middleware")

			sessionID := r.Header.Get("X-Krb-Session")

			// No session at all to verify.
			if sessionID == "" {
				return f(ctx, w, r, request)
			}

			session, err := dbGetSession(ctx, apiImpl.sqlClient, sessionID)
			// Not found among sessions, just continue. Remember this middleware does NOT enforce
			// auth, it only populates metadata.
			if err != nil {
				return f(ctx, w, r, request)
			}

			if time.Now().UnixMilli() > session.Expires {
				return f(ctx, w, r, request)
			}

			ctx = context.WithValue(ctx, adminContextSession, session)
			if session.IsSuper {
				ctx = context.WithValue(ctx, adminContextIsSuperUser, true)
			}

			return f(ctx, w, r, request)
		}
	}
}

func RequireSessionMiddleware() adminapigen.StrictMiddlewareFunc {
	return func(
		f nethttp.StrictHTTPHandlerFunc,
		operationID string,
	) nethttp.StrictHTTPHandlerFunc {
		return func(
			ctx context.Context,
			w http.ResponseWriter,
			r *http.Request,
			request any,
		) (any, error) {
			zerologr.V(20).Info("Running admin require session middleware")

			// auto-approve since no session exists.
			if operationID == "LoginSuperuser" || operationID == "Login" {
				return f(ctx, w, r, request)
			}

			if ContextSessionValid(ctx) {
				return f(ctx, w, r, request)
			}

			return nil, apierror.APIErrNoSession
		}
	}
}

func ContextSessionValid(ctx context.Context) bool {
	val := ctx.Value(adminContextSession)
	if session, ok := val.(*model.Session); ok {
		return time.Now().UnixMilli() <= session.Expires
	}

	// No session found in context, invalid.
	return false
}

func IsSuperUserContext(ctx context.Context) bool {
	val := ctx.Value(adminContextIsSuperUser)
	return val != nil
}
