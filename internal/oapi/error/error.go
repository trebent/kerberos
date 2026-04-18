package apierror

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/trebent/zerologr"
)

type (
	Error struct {
		StatusCode int      `json:"-"`
		Errors     []string `json:"errors"`
	}
)

var (
	_ error = (*Error)(nil)

	ErrUnauthenticated = &Error{
		Errors:     []string{http.StatusText(http.StatusUnauthorized)},
		StatusCode: http.StatusUnauthorized,
	}
	ErrISE = &Error{
		Errors:     []string{http.StatusText(http.StatusInternalServerError)},
		StatusCode: http.StatusInternalServerError,
	}
	ErrNotFound = &Error{
		Errors:     []string{http.StatusText(http.StatusNotFound)},
		StatusCode: http.StatusNotFound,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	ErrMethodNotAllowed = &Error{
		Errors:     []string{http.StatusText(http.StatusMethodNotAllowed)},
		StatusCode: http.StatusMethodNotAllowed,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	ErrForbidden = &Error{
		Errors:     []string{http.StatusText(http.StatusForbidden)},
		StatusCode: http.StatusForbidden,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	ErrUnimplemented = &Error{
		Errors:     []string{"unimplemented"},
		StatusCode: http.StatusNotImplemented,
	}
)

func New(statusCode int, message string) *Error {
	return &Error{StatusCode: statusCode, Errors: []string{message}}
}

func (ae *Error) Error() string {
	return strings.Join(ae.Errors, ", ")
}

func (ae *Error) AsJSON() []byte {
	data, err := json.Marshal(ae)
	if err != nil {
		zerologr.Error(err, "Failed to marshal *apierror.Error")
		return []byte{}
	}
	return data
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
	apiError := &Error{}
	if !errors.As(err, &apiError) {
		zerologr.Info("No error is *Error", "err", err.Error())
		apiError = ErrISE
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiError.StatusCode)
	_, _ = w.Write(apiError.AsJSON())
}
