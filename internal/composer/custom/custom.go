package custom

import (
	"net/http"

	"github.com/trebent/kerberos/internal/composer/types"
)

type custom struct {
	http.Handler

	next types.FlowComponent
}

var _ types.FlowComponent = (*custom)(nil)

func NewComponent(_ ...types.FlowComponent) types.FlowComponent {
	return &custom{}
}

func (c *custom) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c.next.ServeHTTP(w, req)
}

func (c *custom) Next(next types.FlowComponent) {
	c.next = next
}
