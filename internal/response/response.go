// Response package contains common utilities for handling HTTP responses.
package response

import "encoding/json"

type jsonError struct {
	Message string `json:"error"`
}

func JSONError(message string) ([]byte, error) {
	return json.MarshalIndent(jsonError{Message: message}, "", "  ")
}
