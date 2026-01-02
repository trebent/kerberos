//nolint:revive // welp
package types

import "net/http"

type FlowComponent interface {
	http.Handler

	Next(FlowComponent)
}

// Dummy can be used to replace components with a block that just forwards whatever request it's called to handle.
// This is useful to be able to easily disable certain components while keeping the structure the same.
type Dummy struct {
	next FlowComponent
}

var _ FlowComponent = (*Dummy)(nil)

func (d *Dummy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	d.next.ServeHTTP(w, req)
}

func (d *Dummy) Next(next FlowComponent) {
	d.next = next
}
