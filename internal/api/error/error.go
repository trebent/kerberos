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
		StatusCode int      `json:"statusCode"`
		Errors     []string `json:"errors"`
	}
)

var (
	_ error = (*Error)(nil)

	ErrNoPermission = errors.New("you do not have permission to do that")
	ErrNoSession    = errors.New("no session found")
	ErrInternal     = errors.New("internal error")
	ErrNotFound     = errors.New("not found")
	ErrMethod       = errors.New("method not allowed")

	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrNoPermission = &Error{
		Errors:     []string{ErrNoPermission.Error()},
		StatusCode: http.StatusUnauthorized,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrNoSession = &Error{
		Errors:     []string{ErrNoSession.Error()},
		StatusCode: http.StatusUnauthorized,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrInternal = &Error{
		Errors:     []string{ErrInternal.Error()},
		StatusCode: http.StatusInternalServerError,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrNotFound = &Error{
		Errors:     []string{ErrNotFound.Error()},
		StatusCode: http.StatusNotFound,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrMethodNotAllowed = &Error{
		Errors:     []string{ErrMethod.Error()},
		StatusCode: http.StatusMethodNotAllowed,
	}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrForbidden = &Error{
		Errors:     []string{ErrNoPermission.Error()},
		StatusCode: http.StatusForbidden,
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
		apiError = APIErrInternal
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiError.StatusCode)
	_, _ = w.Write(apiError.AsJSON())
}
