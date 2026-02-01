package oas

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestMiddleware(t *testing.T) {
	data, err := os.ReadFile("testspecs/test.yaml")
	if err != nil {
		t.Fatalf("Failed to read data from test OAS: %v", err)
	}

	spec, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		t.Fatalf("Failed to load test OAS: %v", err)
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	handler = ValidationMiddleware(spec)(handler)

	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "/"},
	}

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatal("Did not get expected status code")
	}

	recorder = httptest.NewRecorder()
	req = &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
	}

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatal("Expected bad request from missing body")
	}

	recorder = httptest.NewRecorder()
	req = &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   io.NopCloser(bytes.NewBuffer([]byte(`{"uuid": "uuid"}`))),
	}

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatal("Expected bad request from missing body")
	}
}
