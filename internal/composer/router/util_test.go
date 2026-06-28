package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetBackendName(t *testing.T) {
	tests := []struct {
		name        string
		reqPath     string
		expected    string
		expectError bool
	}{
		{
			name:        "valid backend name",
			reqPath:     "/gw/backend/backend1/some/path",
			expected:    "backend1",
			expectError: false,
		},
		{
			name:        "invalid backend name",
			reqPath:     "/gw/backend//some/path",
			expected:    "",
			expectError: true,
		},
		{
			name:        "no backend name",
			reqPath:     "/gw/backend/",
			expected:    "",
			expectError: true,
		},
		{
			name:        "backend root",
			reqPath:     "/gw/backend/backend1/",
			expected:    "backend1",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.reqPath, nil)
			actual, err := GetBackendName(req)

			if (err != nil) != tt.expectError {
				t.Errorf("GetBackendName() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if actual != tt.expected {
				t.Errorf("GetBackendName() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}
