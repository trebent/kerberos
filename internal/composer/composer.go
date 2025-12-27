package composer

import (
	"net/http"

	"github.com/trebent/kerberos/internal/composer/types"
)

type (
	Opts struct {
		Observability types.FlowComponent
		Router        types.FlowComponent
		Custom        types.FlowComponent
		Forwarder     types.FlowComponent
	}
	impl struct {
		Observability types.FlowComponent
		Router        types.FlowComponent
		Custom        types.FlowComponent
		Forwarder     types.FlowComponent
	}
)

var _ http.Handler = (*impl)(nil)

func New(opts *Opts) http.Handler {
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

func (c *impl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.Observability.ServeHTTP(w, r)
}
