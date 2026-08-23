package lib

import (
	"bytes"
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

type EchoResponse struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body,omitempty"`
}

func Get(url string, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	return Do(req, t, headers...)
}

func ProtectedGet(url string, t *testing.T, session *http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.AddCookie(session)
	return Do(req, t)
}

func Post(url string, body []byte, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	return Do(req, t, headers...)
}

func Put(url string, body []byte, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	return Do(req, t, headers...)
}

func Delete(url string, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	return Do(req, t, headers...)
}

func Patch(url string, body []byte, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	return Do(req, t, headers...)
}

func Trace(url string, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodTrace, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	return Do(req, t, headers...)
}

func Head(url string, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	return Do(req, t, headers...)
}

func Options(url string, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodOptions, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	return Do(req, t, headers...)
}

func Do(req *http.Request, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	for _, h := range headers {
		for key, values := range h {
			req.Header[key] = values
		}
	}
	resp, err := Client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	return resp
}

func VerifyGWResponse(resp *http.Response, expectedCode int, t *testing.T) *EchoResponse {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != expectedCode {
		t.Fatalf("unexpected status code: got %d, want %d", resp.StatusCode, expectedCode)
	}
	response := &EchoResponse{}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return response
}

func VerifyAdminAPIErrorResponse(er *adminapi.APIErrorResponse, t *testing.T) {
	t.Helper()
	if er != nil {
		if len(er.Errors) == 0 {
			t.Fatalf("Expected errors in response body, but got empty errors array")
		}
	} else {
		t.Fatalf("Expected error response but got nil")
	}
}

func VerifyAuthBasicAPIErrorResponse(er *authbasicapi.APIErrorResponse, t *testing.T) {
	t.Helper()
	if er != nil {
		if len(er.Errors) == 0 {
			t.Fatalf("Expected errors in response body, but got empty errors array")
		}
	} else {
		t.Fatalf("Expected error response but got nil")
	}
}

func Matches[T comparable](one, two T, t *testing.T) {
	t.Helper()
	if one != two {
		t.Fatalf("%v is not equal to %v", one, two)
	}
}

func ContainsAll[T comparable](source, reference []T, t *testing.T) {
	t.Helper()
	for _, item := range source {
		if !slices.Contains(reference, item) {
			t.Fatalf("Reference slice does not contain %v", item)
		}
	}
}
