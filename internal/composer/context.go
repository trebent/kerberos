package composer

import "github.com/trebent/kerberos/internal/composer/debug"

type ContextKey string

const (
	// TargetContextKey used by the forwarder to get the target backend.
	TargetContextKey ContextKey = "krb.target"

	// BackendContextKey used to store the backend name.
	BackendContextKey ContextKey = "krb.backend"

	// DebugContextKey used to store the debug call.
	DebugContextKey ContextKey = ContextKey(debug.DebugContextKey)
)
