package composer

import (
	"net/http"
)

type (
	// Composer is an http.Handler that exposes metadata about its FlowComponent chain.
	Composer interface {
		http.Handler

		// GetFlowMeta returns metadata for all FlowComponents in the chain, starting from
		// the first component and traversing to the last.
		GetFlowMeta() *FlowMeta
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

var _ Composer = (*impl)(nil)

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

// GetFlowMeta returns metadata for the entire FlowComponent chain.
func (c *impl) GetFlowMeta() *FlowMeta {
	return c.Observability.GetMeta()
}
