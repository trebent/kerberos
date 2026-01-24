package integration

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

// Validate happy path forwarding to a backend service.
func TestGWHappy(t *testing.T) {
	t.Parallel()

	urlSegment := "/hi"
	url := fmt.Sprintf("http://localhost:%d/gw/backend/echo%s", getPort(), urlSegment)

	response := get(url, t)
	decodedResponse := verifyGWResponse(response, http.StatusOK, t)

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
func TestGWRoot(t *testing.T) {
	t.Parallel()

	testData := "{\"test\": \"value\"}"
	buf := bytes.NewBuffer([]byte(testData))
	urlSegment := "/"
	url := fmt.Sprintf("http://localhost:%d/gw/backend/echo%s", getPort(), urlSegment)

	response := post(url, buf.Bytes(), t)
	decodedResponse := verifyGWResponse(response, http.StatusOK, t)

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
func TestGWNested(t *testing.T) {
	t.Parallel()

	testData := "{\"test\": \"value\"}"
	buf := bytes.NewBuffer([]byte(testData))
	urlSegment := "/hi/hello"
	url := fmt.Sprintf("http://localhost:%d/gw/backend/echo%s", getPort(), urlSegment)

	response := put(url, buf.Bytes(), t)
	decodedResponse := verifyGWResponse(response, http.StatusOK, t)

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
