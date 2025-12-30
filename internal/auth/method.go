package auth

import "net/http"

type Method interface {
	Authenticate(*http.Request) error

	Authenticated(*http.Request) error
	Authorized(*http.Request) error
}
