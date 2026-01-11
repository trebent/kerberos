package method

import "net/http"

type Method interface {
	Authenticated(*http.Request) error
	Authorized(*http.Request) error
}
