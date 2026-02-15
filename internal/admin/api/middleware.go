package adminapi

import (
	"context"
	"net/http"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	adminapigen "github.com/trebent/kerberos/internal/api/admin"
	"github.com/trebent/zerologr"
)

type adminContextKey int

const adminContextIsSuperUser adminContextKey = 0

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
			if session == "" {
				return f(ctx, w, r, request)
			}

			_, ok := apiImpl.superSessions.Load(session)
			if !ok {
				return f(ctx, w, r, request)
			}

			zerologr.Info("Superuser invoker")
			// Found a superuser session, marking the context and forwarding.
			return f(context.WithValue(ctx, adminContextIsSuperUser, true), w, r, request)
		}
	}
}

func IsSuperUserContext(ctx context.Context) bool {
	val := ctx.Value(adminContextIsSuperUser)
	return val != nil
}
