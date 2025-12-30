package auth

import (
	"net/http"

	composertypes "github.com/trebent/kerberos/internal/composer/types"
)

type (
	Authorizer interface {
		composertypes.FlowComponent

		APIEnabled() bool
		RegisterAPI(mux http.Handler)
	}
	authorizer struct {
		next composertypes.FlowComponent
	}
)

var _ Authorizer = (*authorizer)(nil)

func New() composertypes.FlowComponent {
	return &authorizer{}
}

func (a *authorizer) APIEnabled() bool {
	return false
}

func (a *authorizer) RegisterAPI(_ http.Handler) {}

func (a *authorizer) Next(next composertypes.FlowComponent) {
	a.next = next
}

func (a *authorizer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	a.next.ServeHTTP(w, req)
}
