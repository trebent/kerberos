package lib

import (
	"net/http"
	"testing"
)

func CheckErr(err error, t testing.TB) {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func VerifyStatusCode(in int, expected int, t testing.TB) {
	t.Helper()
	if in != expected {
		t.Fatalf("Expected status code %d, got %d", expected, in)
	}
}

func VerifyHeader(headers http.Header, key string, expectedValue string, t testing.TB) {
	t.Helper()
	actualValue := headers.Get(key)
	if actualValue != expectedValue {
		t.Fatalf("Expected header %s to have value %s, got %s", key, expectedValue, actualValue)
	}
}

func VerifyHeaderMissing(headers http.Header, key string, t testing.TB) {
	t.Helper()
	if headers.Get(key) != "" {
		t.Fatalf("Expected header %s to be missing, but it was present", key)
	}
}
