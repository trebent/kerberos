package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSelectCORSMiddleware(t *testing.T) {
	t.Run("allowAll is true", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		handler := SelectCORSMiddleware(nil, true, nextHandler)

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Result().StatusCode)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
			t.Errorf("Expected Access-Control-Allow-Origin header to be set to %s, got %s", "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("allowedOrigins is non-empty", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		allowedOrigins := []string{"http://allowed.com"}
		handler := SelectCORSMiddleware(allowedOrigins, false, nextHandler)

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", "http://allowed.com")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Result().StatusCode)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "http://allowed.com" {
			t.Errorf("Expected Access-Control-Allow-Origin header to be set to %s, got %s", "http://allowed.com", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("neither allowAll nor allowedOrigins is set", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		handler := SelectCORSMiddleware(nil, false, nextHandler)

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Result().StatusCode)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("Expected Access-Control-Allow-Origin header to be empty, got %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}
