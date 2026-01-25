//nolint:revive // welp
package types

type ContextKey string

const (
	// TargetContextKey used by the forwarder to get the target backend.
	TargetContextKey ContextKey = "krb.target"

	// BackendContextKey used to store the backend name.
	BackendContextKey ContextKey = "krb.backend"
)
