package custom

import (
	"net/http"

	"github.com/trebent/kerberos/internal/composer/types"
)

type custom struct {
	handler http.Handler
	next    types.FlowComponent
}

var _ types.FlowComponent = (*custom)(nil)

func NewComponent() types.FlowComponent {
	return &custom{}
}

func (c *custom) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("custom component not implemented")
}

func (c *custom) Compose(next types.FlowComponent) types.FlowComponent {
	panic("custom component not implemented")
}
