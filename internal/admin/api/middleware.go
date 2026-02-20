package adminapi

import (
	"context"
	"net/http"
	"time"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	adminapigen "github.com/trebent/kerberos/internal/api/admin"
	apierror "github.com/trebent/kerberos/internal/api/error"
	"github.com/trebent/zerologr"
)

type adminContextKey int

const (
	adminContextIsSuperUser adminContextKey = 0
	adminContextSession     adminContextKey = 1
)

// AdminSessionMiddleware provides context population of administration session information.
// If put in front of a handler, the context to the handler will contain information that can
// be used to determine if a KRB admin made the call, or if it was external. This is useful
// for internal KRB APIs to verify if a call is external or not.
func AdminSessionMiddleware(
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

			session := r.Header.Get("X-Krb-Session")

			// No session at all to verify.
			if session == "" {
				return f(ctx, w, r, request)
			}

			sessionExpiry, ok := apiImpl.superSessions.Load(session)
			// Not found among super sessions, just continue. Remember this middleware does NOT enforce auth, it
			// only populates metadata.
			if !ok {
				return f(ctx, w, r, request)
			}

			// TODO: This session needs to be set also for other admin users when support for those has been implemented.
			// It's set here not to confuse a non-admin session ID with admin ones, since we only have the super user check
			// right now.

			tsessionExpiry, _ := sessionExpiry.(time.Time)
			if time.Now().Before(tsessionExpiry) {
				ctx = context.WithValue(ctx, adminContextIsSuperUser, true)
				ctx = context.WithValue(ctx, adminContextSession, session)
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
			if operationID == "LoginSuperuser" {
				return f(ctx, w, r, request)
			}

			if ContextSessionValid(ctx) {
				return f(ctx, w, r, request)
			}

			return nil, apierror.APIErrNoSession
		}
	}
}

func SessionIDFromContext(ctx context.Context) string {
	val := ctx.Value(adminContextSession)
	if sval, ok := val.(string); ok {
		return sval
	}

	return ""
}

func ContextSessionValid(ctx context.Context) bool {
	val := ctx.Value(adminContextSession)
	return val != nil
}

func IsSuperUserContext(ctx context.Context) bool {
	val := ctx.Value(adminContextIsSuperUser)
	return val != nil
}
