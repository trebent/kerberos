package composer

import (
	"context"

	"github.com/trebent/kerberos/internal/composer/debug"
)

type ContextKey string

const (
	// TargetContextKey used by the forwarder to get the target backend.
	TargetContextKey ContextKey = "krb.target"

	// BackendContextKey used to store the backend name.
	BackendContextKey ContextKey = "krb.backend"

	// DebugContextKey used to store the debug call.
	DebugContextKey ContextKey = ContextKey(debug.DebugContextKey)
)

// DebugFromContext returns the debug call from the context, or a noop call if none is found.
func DebugFromContext(ctx context.Context) debug.DebuggedCall {
	if ctx == nil {
		return debug.NewNoopCall()
	}

	if call, ok := ctx.Value(DebugContextKey).(debug.DebuggedCall); ok {
		return call
	}

	return debug.NewNoopCall()
}
