package method

import (
	"net/http"
)

type (
	Method interface {
		Authenticated(*http.Request) error
		Authorized(*http.Request) error
	}
	AuthZConfig struct {
		Groups []string            `json:"groups"`
		Paths  map[string][]string `json:"paths"`
	}
)
