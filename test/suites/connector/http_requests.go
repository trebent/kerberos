package connector

import (
	"net/http"
	"testing"
)

func options(url string, t *testing.T, headers ...http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodOptions, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	for _, h := range headers {
		for key, values := range h {
			req.Header[key] = values
		}
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	return resp
}
