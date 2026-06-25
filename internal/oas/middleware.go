package oas

import (
	"context"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/debug"
)

type contextKey string

const debugStartKey contextKey = "oas-debug-start"

// ValidationMiddleware returns a middleware function that will validate incoming requests using the
// input OAS before calling the input next handler.
func ValidationMiddleware(spec *openapi3.T) func(http.Handler) http.Handler {
	// Outer middleware function.
	return func(next http.Handler) http.Handler {
		// options created once per middleware-chain setup, not per request.
		options := &nethttpmiddleware.Options{
			SilenceServersWarning: true,
			DoNotValidateServers:  true,
			Options: openapi3filter.Options{
				AuthenticationFunc: func(context.Context, *openapi3filter.AuthenticationInput) error {
					return nil
				},
			},
			// This is the failure case — transition is logged here and the OAS error
			// handler is called. debugStart is read from the request context so that
			// timing is captured per request, not at middleware setup time.
			ErrorHandlerWithOpts: func(
				ctx context.Context,
				err error,
				w http.ResponseWriter,
				r *http.Request,
				opts nethttpmiddleware.ErrorHandlerOpts,
			) {
				debugStart, _ := r.Context().Value(debugStartKey).(time.Time)
				debugCall := composer.DebugFromContext(r.Context())
				debugCall.AddTransition(
					"oas-validator",
					debug.CallDirectionInbound,
					debugStart,
					time.Now(),
					debug.CallResultFailure,
					err.Error(),
				)

				// DO not finalise the debug call after the error handler is called, the error handler is considered OUTBOUND.
				oasValidationErrorHandler(ctx, err, w, r, opts)
			},
		}

		// Create new handler that intercepts the call before the dispatch to next happens.
		// This is the success case; the failure case is handled by ErrorHandlerWithOpts above.
		actualHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			debugStart, _ := req.Context().Value(debugStartKey).(time.Time)
			debugCall := composer.DebugFromContext(req.Context())
			debugCall.AddTransition(
				"oas-validator",
				debug.CallDirectionInbound,
				debugStart,
				time.Now(),
				debug.CallResultSuccess,
				"",
			)
			next.ServeHTTP(w, req)
		})

		// Build the validator chain once at setup time — not on every request.
		// OapiRequestValidatorWithOptions constructs a gorilla/mux router internally;
		// rebuilding it per request would be a significant performance cost.
		validatorChain := nethttpmiddleware.OapiRequestValidatorWithOptions(
			spec, options,
		)(actualHandler)

		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Inject per-request start time into context so both the success handler
			// and the error handler closure can read accurate timing.
			ctx := context.WithValue(req.Context(), debugStartKey, time.Now())
			validatorChain.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}
