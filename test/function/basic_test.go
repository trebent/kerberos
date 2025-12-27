package ft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

type (
	EchoResponse struct {
		Method  string              `json:"method"`
		URL     string              `json:"url"`
		Headers map[string][]string `json:"headers"`
		Body    []byte              `json:"body"`
	}
)

var (
	port        = 0
	metricsPort = 0
	host        = ""
	client      = &http.Client{}
)

const (
	defaultHost          = "localhost"
	defaultKerberosPort  = 30000
	defaultMetricsPort   = 9464
	defaultClientTimeout = 4 * time.Second
)

func init() {
	client.Timeout = defaultClientTimeout

	val, found := os.LookupEnv("KRB_FT_PORT")
	if !found {
		port = defaultKerberosPort
	}

	decoded, err := strconv.Atoi(val)
	if err != nil {
		port = defaultKerberosPort
	} else {
		port = decoded
	}

	hostVal, found := os.LookupEnv("KRB_FT_HOST")
	if !found {
		host = defaultHost
	} else {
		host = hostVal
	}

	metricsPortVal, found := os.LookupEnv("KRB_FT_METRICS_PORT")
	if !found {
		metricsPort = defaultMetricsPort
	}

	decodedMetricsPort, err := strconv.Atoi(metricsPortVal)
	if err != nil {
		metricsPort = defaultMetricsPort
	} else {
		metricsPort = decodedMetricsPort
	}
}

// Validate happy path forwarding to a backend service.
func TestHappy(t *testing.T) {
	t.Parallel()

	urlSegment := "/hi"
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/gw/backend/echo%s", port, urlSegment), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	response := verifyRespOK(resp, err, t)

	if response.URL != urlSegment {
		t.Errorf("unexpected URL in response: got %s, want %s", response.URL, urlSegment)
	}

	if response.Body != nil {
		t.Errorf("unexpected body in response: got %s, want empty", string(response.Body))
	}

	if response.Method != http.MethodGet {
		t.Errorf("unexpected method in response: got %s, want %s", response.Method, http.MethodGet)
	}
}

// Validate calls to a backend's root path works.
func TestRoot(t *testing.T) {
	t.Parallel()

	testData := "{\"test\": \"value\"}"
	buf := bytes.NewBuffer([]byte(testData))
	urlSegment := "/"
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d/gw/backend/echo%s", port, urlSegment), buf)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	decodedResponse := verifyRespOK(resp, err, t)

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

	for _, val := range resp.Header["Content-Type"] {
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
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:%d/gw/backend/echo%s", port, urlSegment), buf)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	decodedResponse := verifyRespOK(resp, err, t)

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

	for _, val := range resp.Header["Content-Type"] {
		if val != "application/json" {
			t.Errorf("unexpected Content-Type header value: got %s, want %s", val, "application/json")
		}
	}
}

// Validate calls to a non-existent backend yields a not-found status code.
func TestNoBackend(t *testing.T) {
	t.Parallel()

	testData := "{\"test\": \"value\"}"
	buf := bytes.NewBuffer([]byte(testData))
	urlSegment := "/idontexist"
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d/gw/backend%s", port, urlSegment), buf)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	verifyRespStatusCode(resp, err, http.StatusNotFound, t)
}

func verifyRespStatusCode(resp *http.Response, err error, expectedCode int, t *testing.T) {
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedCode {
		t.Fatalf("unexpected status code: got %d, want %d", resp.StatusCode, expectedCode)
	}
}

func verifyRespOK(resp *http.Response, err error, t *testing.T) *EchoResponse {
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	response := &EchoResponse{}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	return response
}
