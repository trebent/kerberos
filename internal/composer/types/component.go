//nolint:revive // welp
package types

import (
	"net/http"

	"github.com/trebent/zerologr"
)

type (
	FlowComponent interface {
		http.Handler

		Next(FlowComponent)
		GetMeta() FlowMeta
	}

	// FlowMeta holds metadata about a FlowComponent and links to the next component's metadata.
	FlowMeta struct {
		// Name is the name of the FlowComponent.
		Name string
		// Description is a short description of what the FlowComponent does.
		Description string
		// Data holds dynamic key-value pairs associated with this component.
		Data map[string]string
		// Next points to the metadata of the subsequent FlowComponent, if any.
		Next *FlowMeta
	}

	// Dummy can be used to replace components with a block that just forwards whatever request it's called to handle.
	// This is useful to be able to easily disable certain components while keeping the structure the same.
	Dummy struct {
		next FlowComponent

		CustomHandler CustomHandlerFunc
		// Returned by Order() int
		O int
	}
	CustomHandlerFunc func(FlowComponent, http.ResponseWriter, *http.Request)
)

var _ FlowComponent = (*Dummy)(nil)

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

func (d *Dummy) GetMeta() FlowMeta {
	meta := FlowMeta{
		Name:        "dummy",
		Description: "A passthrough component that forwards requests to the next component.",
		Data:        map[string]string{},
	}
	if d.next != nil {
		next := d.next.GetMeta()
		meta.Next = &next
	}
	return meta
}

func (d *Dummy) defaultHandler(w http.ResponseWriter, req *http.Request) {
	d.next.ServeHTTP(w, req)
}
