package apierror

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/trebent/zerologr"
)

type (
	Error struct {
		statusCode int

		message string
		wrapped error
	}
)

var (
	_ error = (*Error)(nil)

	ErrNoSession = errors.New("no session found")
	ErrInternal  = errors.New("internal error")
	ErrNotFound  = errors.New("not found")
	ErrMethod    = errors.New("method not allowed")
	ErrForbidden = errors.New("forbidden")

	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrNoSession = &Error{
		message:    ErrNoSession.Error(),
		statusCode: http.StatusUnauthorized,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrInternal = &Error{
		message:    ErrInternal.Error(),
		statusCode: http.StatusInternalServerError,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrNotFound = &Error{
		message:    ErrNotFound.Error(),
		statusCode: http.StatusNotFound,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrMethodNotAllowed = &Error{
		message:    ErrMethod.Error(),
		statusCode: http.StatusMethodNotAllowed,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrForbidden = &Error{
		message:    ErrForbidden.Error(),
		statusCode: http.StatusForbidden,
	}
)

func New(statusCode int, message string) *Error {
	return &Error{statusCode: statusCode, message: message}
}

func (ae *Error) Error() string {
	if ae.wrapped != nil {
		return fmt.Sprintf("%s: %s", ae.message, ae.wrapped.Error())
	}
	return ae.message
}

func (ae *Error) StatusCode() int {
	return ae.statusCode
}

func (ae *Error) Unwrap() error {
	return ae.wrapped
}

func RequestErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	zerologr.Error(err, "Request error")
	ErrorHandler(w, r, err)
}

func ResponseErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	zerologr.Error(err, "Response error")
	ErrorHandler(w, r, err)
}

func ErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	var apiError *Error
	if errors.As(err, &apiError) {
		zerologr.V(20).Info("Was an API error")

		// The first API error will take precedence in status code selection.
		w.WriteHeader(apiError.StatusCode())

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
