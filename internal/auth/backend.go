package auth

import "net/http"

type Backend interface {
	Authenticate(*http.Request) error

	Authenticated(*http.Request) error
	Authorized(*http.Request) error
}
