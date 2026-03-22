package response

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteHeader(t *testing.T) {
	wrapper := &Wrapper{
		responseWriter: httptest.NewRecorder(),
	}

	wrapper.WriteHeader(http.StatusOK)
	wrapper.WriteHeader(http.StatusInternalServerError)

	if wrapper.StatusCode() != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, wrapper.StatusCode())
	}
}

func TestWrite(t *testing.T) {
	wrapper := &Wrapper{
		responseWriter: httptest.NewRecorder(),
	}

	wrapper.Write([]byte("OK"))
	wrapper.WriteHeader(http.StatusInternalServerError)

	if wrapper.StatusCode() != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, wrapper.StatusCode())
	}
}

func TestBodyWrapper(t *testing.T) {
	readCloser := NewBodyWrapper(io.NopCloser(bytes.NewReader([]byte("12345"))))
	bwrapper := readCloser.(*BodyWrapper)

	if bwrapper.NumBytes() != 0 {
		t.Errorf("Expected initial byte count to be 0, got %d", bwrapper.NumBytes())
	}

	chunk := make([]byte, 256)
	_, err := readCloser.Read(chunk)
	if err != io.EOF && err != nil {
		t.Fatalf("Unexpected error during read: %v", err)
	}

	if bwrapper.NumBytes() != 5 {
		t.Errorf("Expected byte count to be 5, got %d", bwrapper.NumBytes())
	}
}
