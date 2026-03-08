package composer

import (
	"net/http"

	"github.com/trebent/zerologr"
)

type (
	FlowComponent interface {
		http.Handler

		Next(FlowComponent)
		GetMeta() []*FlowMeta
	}

	Target interface {
		Host() string
		Port() int
	}

	// FlowMeta holds metadata about a FlowComponent and links to the next component's metadata.
	FlowMeta struct {
		// Name is the name of the FlowComponent.
		Name string `json:"name"`
		// Data holds dynamic key-value pairs associated with this component.
		Data map[string]any `json:"data"`
	}

	// Dummy can be used to replace components with a block that just forwards whatever request it's called to handle.
	// This is useful to be able to easily disable certain components while keeping the structure the same.
	Dummy struct {
		next FlowComponent

		CustomHandler CustomHandlerFunc
		// Returned by Order() int. Don't import the custom types package here to avoid circular
		// imports, so we just use an int and interpret it as the order.
		O int
	}
	CustomHandlerFunc func(FlowComponent, http.ResponseWriter, *http.Request)
)

var _ FlowComponent = (*Dummy)(nil)

const (
	MetaKeyOrder = "order"
)

func (d *Dummy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	zerologr.V(20).Info("Dummy component handling call to " + req.Method + " " + req.URL.Path)

	if d.CustomHandler != nil {
		d.CustomHandler(d.next, w, req)
	} else {
		d.defaultHandler(w, req)
	}
}

func (d *Dummy) Next(next FlowComponent) {
	d.next = next
}

func (d *Dummy) Order() int {
	return d.O
}

func (d *Dummy) GetMeta() []*FlowMeta {
	meta := &FlowMeta{
		Name: "dummy",
		Data: map[string]any{},
	}
	if d.next != nil {
		return append([]*FlowMeta{meta}, d.next.GetMeta()...)
	}
	return []*FlowMeta{meta}
}

func (d *Dummy) defaultHandler(w http.ResponseWriter, req *http.Request) {
	d.next.ServeHTTP(w, req)
}
