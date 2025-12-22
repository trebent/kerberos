package composer

import (
	"net/http"

	"github.com/trebent/kerberos/internal/composer/types"
)

type (
	Composer struct {
		Observability types.FlowComponent
		Router        types.FlowComponent
		Custom        types.FlowComponent
		Forwarder     types.FlowComponent
	}
)

var _ http.Handler = (*Composer)(nil)

func (c *Composer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	/*
		Each composer component will run one-after-another in a pipeline fashion.
		Each component can decide to short-circuit the request by returning an error.
		If no error is returned, the next component in the pipeline is executed.

		The order of execution is:
		1. Observability
		2. Router
		3. Composable
		4. Forwarder

		If any component errors out it can simply modify the request to return the error to the client.
		To set up the pipeline, each component is passed the next component in line. If a component fails,
		it can return an error which will stop the pipeline execution.
	*/

	// handler w, r -> obs calls itself, then calls the next which is router
	c.Observability.Compose(c.Router.Compose(c.Custom.Compose(c.Forwarder))).ServeHTTP(w, r)
}
