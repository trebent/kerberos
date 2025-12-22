package types

import "net/http"

type FlowComponent interface {
	http.Handler

	Compose(next FlowComponent) FlowComponent
}
