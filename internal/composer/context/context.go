package composerctx

type ContextKey string

const (
	// TargetContextKey used by the forwarder to get the target backend.
	TargetContextKey ContextKey = "target"

	// BackendContextKey used to store the backend name for OTEL purposes.
	BackendContextKey ContextKey = "krb.backend"
)
