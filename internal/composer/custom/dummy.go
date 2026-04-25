package custom

import (
	"net/http"

	"github.com/trebent/kerberos/internal/composer"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	"github.com/trebent/zerologr"
)

type (
	// Dummy is a duplicate of composer.Dummy, but with the addition of the Order() int method to be able to be used in
	// the custom component chain. It can also be used as a simple dummy component in other contexts.
	Dummy struct {
		next composer.FlowComponent

		CustomHandler composer.CustomHandlerFunc
		O             int
	}
)

var (
	_ composer.FlowComponent = (*Dummy)(nil)
	_ Ordered                = (*Dummy)(nil)
)

func (d *Dummy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if zerologr.V(20).Enabled() {
		zerologr.Info("Dummy component handling call to " + req.Method + " " + req.URL.Path)
	}

	if d.CustomHandler != nil {
		d.CustomHandler(d.next, w, req)
	} else {
		d.defaultHandler(w, req)
	}
}

func (d *Dummy) Next(next composer.FlowComponent) {
	d.next = next
}

func (d *Dummy) Order() int {
	return d.O
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
