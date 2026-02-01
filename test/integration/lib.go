package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	authadminapi "github.com/trebent/kerberos/ft/client/auth/admin"
	authbasicapi "github.com/trebent/kerberos/ft/client/auth/basic"
)

type (
	EchoResponse struct {
		Method  string              `json:"method"`
		URL     string              `json:"url"`
		Headers map[string][]string `json:"headers"`
		Body    json.RawMessage     `json:"body,omitempty"`
	}
	RequestEditorFn func(ctx context.Context, req *http.Request) error
)

var (
	client = &http.Client{Timeout: 4 * time.Second}

	basicAuthClient, _ = authbasicapi.NewClientWithResponses(
		fmt.Sprintf("http://%s:%d/api/auth/basic", getHost(), getPort()),
	)
	adminClient, _ = authadminapi.NewClientWithResponses(
		fmt.Sprintf("http://%s:%d/api/auth/admin", getHost(), getPort()),
	)
)

const (
	defaultHost              = "localhost"
	defaultKerberosPort      = 30000
	defaultMetricsPort       = 9464
	defaultJaegerReadAPIPort = 16685
)

func get(url string, t *testing.T, headers ...http.Header) *http.Response {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func post(url string, body []byte, t *testing.T, headers ...http.Header) *http.Response {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func put(url string, body []byte, t *testing.T, headers ...http.Header) *http.Response {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func delete(url string, t *testing.T, headers ...http.Header) *http.Response {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func patch(url string, body []byte, t *testing.T, headers ...http.Header) *http.Response {
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	return do(req, t, headers...)
}

func do(req *http.Request, t *testing.T, headers ...http.Header) *http.Response {
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

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func verifyStatusCode(in int, expected int, t *testing.T) {
	if in != expected {
		t.Fatalf("Expected status code %d, got %d", expected, in)
	}
}

func verifyGWResponse(resp *http.Response, expectedCode int, t *testing.T) *EchoResponse {
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

func matches[T comparable](one, two T, t *testing.T) {
	if one != two {
		t.Fatalf("%v is not equal to %v", one, two)
	}
}

func containsAll[T comparable](source, reference []T, t *testing.T) {
	for _, item := range source {
		if !slices.Contains(reference, item) {
			t.Fatalf("Reference slice does not contain %v", item)
		}
	}
}

func requestEditorSessionID(sessionID string) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("x-krb-session", sessionID)
		return nil
	}
}

func getPort() int {
	val, found := os.LookupEnv("KRB_FT_PORT")
	if !found {
		return defaultKerberosPort
	}

	decoded, err := strconv.Atoi(val)
	if err != nil {
		return defaultKerberosPort
	} else {
		return decoded
	}
}

func getHost() string {
	hostVal, found := os.LookupEnv("KRB_FT_HOST")
	if !found {
		return defaultHost
	} else {
		return hostVal
	}
}

func getMetricsPort() int {
	metricsPortVal, found := os.LookupEnv("KRB_FT_METRICS_PORT")
	if !found {
		return defaultMetricsPort
	}

	decodedMetricsPort, err := strconv.Atoi(metricsPortVal)
	if err != nil {
		return defaultMetricsPort
	} else {
		return decodedMetricsPort
	}
}

func getJaegerAPIPort() int {
	jaegerPortVal, found := os.LookupEnv("KRB_FT_JAEGER_PORT")
	if !found {
		return defaultJaegerReadAPIPort
	}

	decodedJaegerPort, err := strconv.Atoi(jaegerPortVal)
	if err != nil {
		return defaultJaegerReadAPIPort
	} else {
		return decodedJaegerPort
	}
}

func extractSession(resp *http.Response, t *testing.T) string {
	session := resp.Header.Get("x-krb-session")
	if session == "" {
		t.Fatalf("missing session header in response")
	}

	return session
}
