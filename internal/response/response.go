// Package response contains common utilities for handling HTTP responses.
package response

import (
	"encoding/json"
	"net/http"
)

type jsonError struct {
	Message string `json:"error"`
}

func JSONError(w http.ResponseWriter, err error, statusCode int) {
	errorData, _ := json.MarshalIndent(jsonError{Message: err.Error()}, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(errorData)
}
