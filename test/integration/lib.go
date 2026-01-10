package integration

import (
	"bytes"
	"net/http"
	"slices"
	"testing"
	"time"
)

var (
	client = &http.Client{Timeout: 4 * time.Second}
)

func get(url string, t *testing.T) *http.Response {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	return resp
}

func post(url string, body []byte, t *testing.T) *http.Response {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	return resp
}

func put(url string, body []byte, t *testing.T) *http.Response {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	return resp
}

func delete(url string, t *testing.T) *http.Response {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	return resp
}

func patch(url string, body []byte, t *testing.T) *http.Response {
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
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
