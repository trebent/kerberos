package integration

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

func get(url string, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func protectedGet(url string, t *testing.T, session *http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.AddCookie(session)

	return do(req, t)
}

func post(url string, body []byte, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func put(url string, body []byte, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func delete(url string, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func patch(url string, body []byte, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func trace(url string, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodTrace, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func head(url string, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func options(url string, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodOptions, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func do(req *http.Request, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	for _, headers := range headers {
		for key, values := range headers {
			req.Header[key] = values
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	return resp
}

func verifyGWResponse(resp *http.Response, expectedCode int, t *testing.T) *EchoResponse {
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

func verifyAdminAPIErrorResponse(er *adminapi.APIErrorResponse, t *testing.T) {
	t.Helper()
	if er != nil {
		if len(er.Errors) == 0 {
			t.Fatalf("Expected errors in response body, but got empty errors array")
		}
	} else {
		t.Fatalf("Expected error response but got nil")
	}
}

func verifyAuthBasicAPIErrorResponse(er *authbasicapi.APIErrorResponse, t *testing.T) {
	t.Helper()
	if er != nil {
		if len(er.Errors) == 0 {
			t.Fatalf("Expected errors in response body, but got empty errors array")
		}
	} else {
		t.Fatalf("Expected error response but got nil")
	}
}

func matches[T comparable](one, two T, t *testing.T) {
	t.Helper()
	if one != two {
		t.Fatalf("%v is not equal to %v", one, two)
	}
}

func containsAll[T comparable](source, reference []T, t *testing.T) {
	t.Helper()
	for _, item := range source {
		if !slices.Contains(reference, item) {
			t.Fatalf("Reference slice does not contain %v", item)
		}
	}
}
