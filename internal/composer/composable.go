package composer

import "net/http"

type (
	composable struct{}
)

var _ FlowComponent = (*composable)(nil)

// ServeHTTP implements [FlowComponent].
func (c *composable) ServeHTTP(http.ResponseWriter, *http.Request) {
	panic("unimplemented")
}

func (c *composable) Compose(next FlowComponent) FlowComponent {
	return c
}
