package oas

import (
	"context"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/trebent/zerologr"
)

func ValidationMiddleware(spec *openapi3.T) nethttp.StrictHTTPMiddlewareFunc {
	options := &nethttpmiddleware.Options{
		SilenceServersWarning: true,
		DoNotValidateServers:  true,
		ErrorHandlerWithOpts:  oasValidationErrorHandler,
		Options: openapi3filter.Options{
			AuthenticationFunc: func(context.Context, *openapi3filter.AuthenticationInput) error { return nil },
		},
	}
	mw := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, options)
	// Adapt a nethttp middleware (func(http.Handler) http.Handler) into
	// an oapi-codegen StrictMiddlewareFunc. The middleware will call the
	// provided "next" http.Handler only when validation succeeds; when it
	// doesn't, the middleware is expected to write a response itself.
	adapter := func(m func(http.Handler) http.Handler) nethttp.StrictHTTPMiddlewareFunc {
		return func(next nethttp.StrictHTTPHandlerFunc, _ string) nethttp.StrictHTTPHandlerFunc {
			return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
				zerologr.V(20).Info("Running OAS validation middleware", "url", r.URL.Path)

				var called bool
				var res any
				var resErr error

				// nextHandler will be invoked by the nethttp middleware only when
				// validation passes. We capture the result of calling the strict
				// handler and propagate it back.
				nextHandler := http.HandlerFunc(func(w2 http.ResponseWriter, r2 *http.Request) {
					called = true
					res, resErr = next(ctx, w2, r2, request)
				})

				m(nextHandler).ServeHTTP(w, r)

				// If the middleware never called our next handler, it already
				// wrote the response (e.g. validation failed). In that case we
				// return nil, nil so the strict handler does not attempt to
				// write anything else.
				if !called {
					return nil, nil
				}

				return res, resErr
			}
		}
	}

	return adapter(mw)
}
