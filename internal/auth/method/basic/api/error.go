package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/trebent/zerologr"
)

type (
	Error struct {
		StatusCode int

		message string
		wrapped error
	}
)

var _ error = (*Error)(nil)

func (ae *Error) Error() string {
	if ae.wrapped != nil {
		return fmt.Sprintf("%s: %v", ae.message, ae.wrapped)
	}
	return ae.message
}

func (ae *Error) Unwrap() error {
	return ae.wrapped
}

func RequestErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	zerologr.V(20).Info("Running request error handler")
	errorHandler(w, r, err)
}

func ResponseErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	zerologr.V(20).Info("Running response error handler")
	errorHandler(w, r, err)
}

func errorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	var apiError *Error
	if errors.As(err, &apiError) {
		zerologr.V(20).Info("Was an API error")

		// The first API error will take precedence in status code selection.
		w.WriteHeader(apiError.StatusCode)

		builder := strings.Builder{}
		_, _ = builder.WriteString("{\"errors\": [")
		for err := apiError.Unwrap(); err != nil; err = errors.Unwrap(err) {
			// Verify all unwrapped errors are api.Errors, if not, discard.
			apiErr := &Error{}
			if errors.As(err, &apiErr) {
				zerologr.Info("Discarded non *Error from response", "err", err.Error())
				continue
			}
			_, _ = fmt.Fprintf(&builder, "  \"%s\"", err.Error())
		}
		_, _ = builder.WriteString("]}")
		_, _ = w.Write([]byte(builder.String()))
	} else {
		zerologr.Info("No error is *Error", "err", err.Error())

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "{\"errors\": [\"%s\"]}", ErrInternal.Error())
	}
}
