package custom

import (
	"net/http"

	"github.com/trebent/kerberos/internal/composer/types"
)

type custom struct {
	http.Handler
	types.FlowComponent
}

var _ types.FlowComponent = (*custom)(nil)

func NewComponent() types.FlowComponent {
	return &custom{}
}

func (c *custom) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
	panic("custom component not implemented")
}

func (c *custom) Next(_ types.FlowComponent) {
	panic("custom component not implemented")
}
