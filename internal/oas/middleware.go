package oas

import (
	"context"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

// ValidationMiddleware returns a wrapper function that will validate incoming requests using the
// input OAS before calling the input next handler.
func ValidationMiddleware(spec *openapi3.T) func(http.Handler) http.Handler {
	options := &nethttpmiddleware.Options{
		SilenceServersWarning: true,
		DoNotValidateServers:  true,
		ErrorHandlerWithOpts:  oasValidationErrorHandler,
		Options: openapi3filter.Options{
			AuthenticationFunc: func(context.Context, *openapi3filter.AuthenticationInput) error {
				return nil
			},
		},
	}
	return nethttpmiddleware.OapiRequestValidatorWithOptions(spec, options)
}
