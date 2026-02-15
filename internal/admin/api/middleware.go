package adminapi

import (
	"context"
	"net/http"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	adminapigen "github.com/trebent/kerberos/internal/api/admin"
)

type adminContextKey int

const adminContextIsSuperUser adminContextKey = 0

func AdminMiddleware(
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
			session := r.Header.Get("X-Krb-Session")
			if session == "" {
				return f(ctx, w, r, request)
			}

			_, ok := apiImpl.superSessions.Load(session)
			if !ok {
				return f(ctx, w, r, request)
			}

			// Found a superuser session, marking the context and forwarding.
			return f(context.WithValue(ctx, adminContextIsSuperUser, true), w, r, request)
		}
	}
}

func IsSuperUserContext(ctx context.Context) bool {
	val := ctx.Value(adminContextIsSuperUser)
	return val != nil
}
