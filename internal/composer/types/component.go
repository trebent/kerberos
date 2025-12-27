//nolint:revive // welp
package types

import "net/http"

type FlowComponent interface {
	http.Handler

	Next(FlowComponent)
}
