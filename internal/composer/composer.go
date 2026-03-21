package composer

import (
	"net/http"

	adminext "github.com/trebent/kerberos/internal/admin/extensions"
	adminapi "github.com/trebent/kerberos/internal/api/admin"
)

type (
	// Composer is an http.Handler that exposes metadata about its FlowComponent chain.
	Composer interface {
		http.Handler

		// GetFlow implements the admin extension for flow inspection.
		GetFlow() []adminapi.FlowMeta
	}
	Opts struct {
		Observability FlowComponent
		Router        FlowComponent
		Custom        FlowComponent
		Forwarder     FlowComponent
	}
	impl struct {
		Observability FlowComponent
		Router        FlowComponent
		Custom        FlowComponent
		Forwarder     FlowComponent
	}
)

var (
	_ Composer             = (*impl)(nil)
	_ adminext.FlowFetcher = (*impl)(nil)
)

func New(opts *Opts) Composer {
	opts.Observability.Next(opts.Router)
	opts.Router.Next(opts.Custom)
	opts.Custom.Next(opts.Forwarder)

	return &impl{
		Observability: opts.Observability,
		Router:        opts.Router,
		Custom:        opts.Custom,
		Forwarder:     opts.Forwarder,
	}
}

func (c *impl) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c.Observability.ServeHTTP(w, req)
}

// GetFlow returns metadata for the entire FlowComponent chain.
func (c *impl) GetFlow() []adminapi.FlowMeta {
	return c.Observability.GetMeta()
}
