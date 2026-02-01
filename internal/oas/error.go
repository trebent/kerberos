package oas

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	apierror "github.com/trebent/kerberos/internal/api/error"
	"github.com/trebent/zerologr"
)

func oasValidationErrorHandler(
	ctx context.Context,
	err error,
	w http.ResponseWriter,
	req *http.Request,
	opts nethttpmiddleware.ErrorHandlerOpts,
) {
	logger, logrErr := logr.FromContext(ctx)
	if logrErr != nil {
		logger = zerologr.WithName("oas-validator")
	} else {
		logger = logger.WithName("oas-validator")
	}

	logger.Error(err, "OAS validation failed", "path", req.URL.Path)
	apierror.ErrorHandler(w, req, apierror.New(opts.StatusCode, err.Error()))
}
