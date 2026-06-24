package debug

import (
	"context"
)

type (
	dummy struct {
		returnedCall DebuggedCall
	}
)

var _ Debugger = (*dummy)(nil)

// NewDummy returns a new instance of a dummy Debugger that does not perform any actual debugging.
// This is useful for testing or when debugging is not required.
func NewDummy(returnedCall DebuggedCall) Debugger {
	return &dummy{returnedCall: returnedCall}
}

// Start implements [Debugger].
func (d *dummy) Start(context.Context) (DebuggedCall, context.Context) {
	//nolint:revive,staticcheck // intentional
	return noop, context.WithValue(context.Background(), DebugContextKey, noop)
}
