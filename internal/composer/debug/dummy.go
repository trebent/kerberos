package debug

import (
	"context"
)

type (
	dummy struct {
		returnedCall Call
	}
)

var _ Debugger = (*dummy)(nil)

// NewDummy returns a new instance of a dummy Debugger that does not perform any actual debugging.
// This is useful for testing or when debugging is not required.
func NewDummy(returnedCall Call) Debugger {
	return &dummy{returnedCall: returnedCall}
}

// Debug implements [Debugger].
func (d *dummy) Debug(ctx context.Context) (Call, context.Context) {
	//nolint:revive,staticcheck // intentional
	return noop, context.WithValue(ctx, DebugContextKey, noop)
}
