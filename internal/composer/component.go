package composer

import (
	"net/http"

	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	"github.com/trebent/zerologr"
)

type (
	FlowComponent interface {
		http.Handler

		Next(FlowComponent)
		GetMeta() []adminapi.FlowMeta
	}

	// FlowMeta holds metadata about a FlowComponent.
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

func (d *Dummy) GetMeta() []adminapi.FlowMeta {
	fmd := adminapi.FlowMeta_Data{}
	if err := fmd.FromNoFlowMetaData(adminapi.NoFlowMetaData{}); err != nil {
		panic(err)
	}

	meta := adminapi.FlowMeta{
		Name: "dummy",
		Data: fmd,
	}
	if d.next != nil {
		return append([]adminapi.FlowMeta{meta}, d.next.GetMeta()...)
	}
	return []adminapi.FlowMeta{meta}
}

func (d *Dummy) defaultHandler(w http.ResponseWriter, req *http.Request) {
	d.next.ServeHTTP(w, req)
}
