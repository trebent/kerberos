package oas

import (
	"net/http"

	"github.com/trebent/kerberos/internal/composer/custom"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
	"github.com/trebent/kerberos/internal/config"
)

type (
	validator struct {
		next composertypes.FlowComponent
		cfg  *oasConfig
	}
	Opts struct {
		Cfg config.Map

		// TODO: use this to register API documentation.
		Mux *http.ServeMux
	}
)

var (
	_ composertypes.FlowComponent = &validator{}
	_ custom.Ordered              = &validator{}
)

func New(opts *Opts) composertypes.FlowComponent {
	cfg := config.AccessAs[*oasConfig](opts.Cfg, configName)
	return &validator{cfg: cfg}
}

func (v *validator) Order() int {
	return v.cfg.Order
}

// Next implements [types.FlowComponent].
func (v *validator) Next(next composertypes.FlowComponent) {
	v.next = next
}

// ServeHTTP implements [types.FlowComponent].
func (v *validator) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// TODO: do validation here.

	v.next.ServeHTTP(w, req)
}
