package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// Validate happy path forwarding for all HTTP methods.
func TestGWHappy(t *testing.T) {
	t.Parallel()

	baseURL := fmt.Sprintf("http://localhost:%d/gw/backend/echo", getPort())
	bodyData := []byte(`{"test": "value"}`)

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{name: "GET", method: http.MethodGet, path: "/hi"},
		{name: "POST", method: http.MethodPost, path: "/", body: bodyData},
		{name: "PUT", method: http.MethodPut, path: "/long/hello", body: bodyData},
		{name: "DELETE", method: http.MethodDelete, path: "/hi"},
		{name: "PATCH", method: http.MethodPatch, path: "/hi", body: bodyData},
		{name: "TRACE", method: http.MethodTrace, path: "/hi"},
		{name: "OPTIONS", method: http.MethodOptions, path: "/hi"},
		{name: "HEAD", method: http.MethodHead, path: "/hi"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url := baseURL + tc.path

			// HEAD responses carry no body; verify the status code only.
			if tc.method == http.MethodHead {
				response := head(url, t)
				defer response.Body.Close()
				if response.StatusCode != http.StatusOK {
					t.Fatalf("unexpected status code: got %d, want %d", response.StatusCode, http.StatusOK)
				}
				return
			}

			var req *http.Request
			var err error
			if tc.body != nil {
				req, err = http.NewRequest(tc.method, url, bytes.NewBuffer(tc.body))
			} else {
				req, err = http.NewRequest(tc.method, url, nil)
			}
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			response := do(req, t)
			decoded := verifyGWResponse(response, http.StatusOK, t)

			if decoded.URL != tc.path {
				t.Errorf("unexpected URL in response: got %s, want %s", decoded.URL, tc.path)
			}
			if decoded.Method != tc.method {
				t.Errorf("unexpected method in response: got %s, want %s", decoded.Method, tc.method)
			}
			if tc.body != nil {
				if decoded.Body == nil {
					t.Errorf("unexpected body in response: got nil, want non-empty")
					return
				}
				m := map[string]any{}
				if err := json.Unmarshal(decoded.Body, &m); err != nil {
					t.Errorf("failed to unmarshal response body: %v", err)
				}
				if m["test"] != "value" {
					t.Errorf("unexpected body in response: got %s, want %s", string(decoded.Body), string(bodyData))
				}
			} else {
				if decoded.Body != nil {
					t.Errorf("unexpected body in response: got %s, want empty", string(decoded.Body))
				}
			}
		})
	}
}

// Validate calls to a non-existent backend yields a not-found status code.
func TestGWNoBackend(t *testing.T) {
	t.Parallel()

	testData := "{\"test\": \"value\"}"
	urlSegment := "/idontexist/"
	url := fmt.Sprintf("http://localhost:%d/gw/backend%s", getPort(), urlSegment)
	t.Logf("Sending to non-existent backend url %s", url)
	response := post(url, []byte(testData), t)

	_ = verifyGWResponse(response, http.StatusNotFound, t)
}

func TestGWBackendFormat(t *testing.T) {
	t.Parallel()

	testData := "{\"test\": \"value\"}"
	url := fmt.Sprintf("http://localhost:%d/gw/back", getPort())
	t.Logf("Sending to funky url %s", url)
	response := post(url, []byte(testData), t)

	_ = verifyGWResponse(response, http.StatusBadRequest, t)
}
