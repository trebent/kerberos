package ft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

type (
	EchoResponse struct {
		Method  string              `json:"method"`
		URL     string              `json:"url"`
		Headers map[string][]string `json:"headers"`
		Body    []byte              `json:"body"`
	}
)

// Validate happy path forwarding to a backend service.
func TestHappy(t *testing.T) {
	t.Parallel()

	urlSegment := "/hi"
	url := fmt.Sprintf("http://localhost:%d/gw/backend/echo%s", port, urlSegment)

	response := get(url, t)
	decodedResponse := verifyResponse(response, http.StatusOK, t)

	if decodedResponse.URL != urlSegment {
		t.Errorf("unexpected URL in response: got %s, want %s", decodedResponse.URL, urlSegment)
	}

	if decodedResponse.Body != nil {
		t.Errorf("unexpected body in response: got %s, want empty", string(decodedResponse.Body))
	}

	if decodedResponse.Method != http.MethodGet {
		t.Errorf("unexpected method in response: got %s, want %s", decodedResponse.Method, http.MethodGet)
	}
}

// Validate calls to a backend's root path works.
func TestRoot(t *testing.T) {
	t.Parallel()

	testData := "{\"test\": \"value\"}"
	buf := bytes.NewBuffer([]byte(testData))
	urlSegment := "/"
	url := fmt.Sprintf("http://localhost:%d/gw/backend/echo%s", port, urlSegment)

	response := post(url, buf.Bytes(), t)
	decodedResponse := verifyResponse(response, http.StatusOK, t)

	if decodedResponse.URL != urlSegment {
		t.Errorf("unexpected URL in response: got %s, want %s", decodedResponse.URL, urlSegment)
	}

	if decodedResponse.Body == nil {
		t.Errorf("unexpected body in response: got %s, want non-empty", string(decodedResponse.Body))
	}

	if decodedResponse.Method != http.MethodPost {
		t.Errorf("unexpected method in response: got %s, want %s", decodedResponse.Method, http.MethodPost)
	}

	if string(decodedResponse.Body) != testData {
		t.Errorf("unexpected body in response: got %s, want %s", string(decodedResponse.Body), testData)
	}

	for _, val := range response.Header["Content-Type"] {
		if val != "application/json" {
			t.Errorf("unexpected Content-Type header value: got %s, want %s", val, "application/json")
		}
	}
}

// Validate calls to a backend's nested path works.
func TestNested(t *testing.T) {
	t.Parallel()

	testData := "{\"test\": \"value\"}"
	buf := bytes.NewBuffer([]byte(testData))
	urlSegment := "/soi/mae"
	url := fmt.Sprintf("http://localhost:%d/gw/backend/echo%s", port, urlSegment)

	response := put(url, buf.Bytes(), t)
	decodedResponse := verifyResponse(response, http.StatusOK, t)

	if decodedResponse.URL != urlSegment {
		t.Errorf("unexpected URL in response: got %s, want %s", decodedResponse.URL, urlSegment)
	}

	if decodedResponse.Body == nil {
		t.Errorf("unexpected body in response: got %s, want non-empty", string(decodedResponse.Body))
	}

	if decodedResponse.Method != http.MethodPut {
		t.Errorf("unexpected method in response: got %s, want %s", decodedResponse.Method, http.MethodPut)
	}

	if string(decodedResponse.Body) != testData {
		t.Errorf("unexpected body in response: got %s, want %s", string(decodedResponse.Body), testData)
	}

	for _, val := range response.Header["Content-Type"] {
		if val != "application/json" {
			t.Errorf("unexpected Content-Type header value: got %s, want %s", val, "application/json")
		}
	}
}

// Validate calls to a non-existent backend yields a not-found status code.
func TestNoBackend(t *testing.T) {
	t.Parallel()

	testData := "{\"test\": \"value\"}"
	urlSegment := "/idontexist"
	url := fmt.Sprintf("http://localhost:%d/gw/backend%s", port, urlSegment)
	response := post(url, []byte(testData), t)

	_ = verifyResponse(response, http.StatusNotFound, t)
}

func verifyResponse(resp *http.Response, expectedCode int, t *testing.T) *EchoResponse {
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
