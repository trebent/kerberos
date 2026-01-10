//nolint:revive // welp
package types

import "net/http"

type (
	FlowComponent interface {
		http.Handler

		Next(FlowComponent)
	}

	// Dummy can be used to replace components with a block that just forwards whatever request it's called to handle.
	// This is useful to be able to easily disable certain components while keeping the structure the same.
	Dummy struct {
		next FlowComponent

		CustomHandler CustomHandlerFunc
	}
	CustomHandlerFunc func(FlowComponent, http.ResponseWriter, *http.Request)
)

var _ FlowComponent = (*Dummy)(nil)

func (d *Dummy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if d.CustomHandler != nil {
		d.CustomHandler(d.next, w, req)
	} else {
		d.defaultHandler(w, req)
	}
}

func (d *Dummy) Next(next FlowComponent) {
	d.next = next
}

func (d *Dummy) defaultHandler(w http.ResponseWriter, req *http.Request) {
	d.next.ServeHTTP(w, req)
}
