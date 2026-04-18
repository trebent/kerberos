package apierror_test

import (
	"errors"
	"net/http/httptest"
	"testing"

	apierror "github.com/trebent/kerberos/internal/oapi/error"
)

func TestResponseHandler(t *testing.T) {
	recorder := httptest.NewRecorder()

	apierror.ResponseErrorHandler(recorder, nil, apierror.ErrNotFound)

	if recorder.Code != apierror.ErrNotFound.StatusCode {
		t.Errorf("expected status code %d, got %d", apierror.ErrNotFound.StatusCode, recorder.Code)
	}

	if recorder.Body.String() != string(apierror.ErrNotFound.AsJSON()) {
		t.Errorf("expected body %s, got %s", string(apierror.ErrNotFound.AsJSON()), recorder.Body.String())
	}
}

func TestResponseHandler_UnknownError(t *testing.T) {
	recorder := httptest.NewRecorder()

	apierror.ResponseErrorHandler(recorder, nil, errors.New("unknown error"))

	if recorder.Code != apierror.ErrISE.StatusCode {
		t.Errorf("expected status code %d, got %d", apierror.ErrISE.StatusCode, recorder.Code)
	}

	if recorder.Body.String() != string(apierror.ErrISE.AsJSON()) {
		t.Errorf("expected body %s, got %s", string(apierror.ErrISE.AsJSON()), recorder.Body.String())
	}
}

func TestRequestHandler(t *testing.T) {
	recorder := httptest.NewRecorder()

	apierror.RequestErrorHandler(recorder, nil, apierror.ErrForbidden)

	if recorder.Code != apierror.ErrForbidden.StatusCode {
		t.Errorf("expected status code %d, got %d", apierror.ErrForbidden.StatusCode, recorder.Code)
	}

	if recorder.Body.String() != string(apierror.ErrForbidden.AsJSON()) {
		t.Errorf("expected body %s, got %s", string(apierror.ErrForbidden.AsJSON()), recorder.Body.String())
	}
}

func TestRequestHandler_UnknownError(t *testing.T) {
	recorder := httptest.NewRecorder()

	apierror.RequestErrorHandler(recorder, nil, errors.New("unknown error"))

	if recorder.Code != apierror.ErrISE.StatusCode {
		t.Errorf("expected status code %d, got %d", apierror.ErrISE.StatusCode, recorder.Code)
	}

	if recorder.Body.String() != string(apierror.ErrISE.AsJSON()) {
		t.Errorf("expected body %s, got %s", string(apierror.ErrISE.AsJSON()), recorder.Body.String())
	}
}
