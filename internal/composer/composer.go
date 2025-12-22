package composer

import "net/http"

type Composer struct {
	Observability http.Handler
	Router        http.Handler
	Composable    http.Handler
	Forwarder     http.Handler
}

var _ http.Handler = (*Composer)(nil)

func (c *Composer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.Observability != nil {
		c.Observability.ServeHTTP(w, r)
		return
	}
	if c.Router != nil {
		c.Router.ServeHTTP(w, r)
		return
	}
	if c.Composable != nil {
		c.Composable.ServeHTTP(w, r)
		return
	}
	if c.Forwarder != nil {
		c.Forwarder.ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}
