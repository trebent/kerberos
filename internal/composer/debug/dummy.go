package debug

import (
	"context"
	"time"
)

type (
	dummy struct{}
	noop  struct{}
)

var (
	_ Debugger     = (*dummy)(nil)
	_ DebuggedCall = (*noop)(nil)

	n = &noop{}
)

// NewDummy returns a new instance of a dummy Debugger that does not perform any actual debugging.
// This is useful for testing or when debugging is not required.
func NewDummy() Debugger {
	return &dummy{}
}

// Start implements [Debugger].
func (d *dummy) Start(context.Context) (DebuggedCall, context.Context) {
	//nolint:revive,staticcheck // intentional
	return n, context.WithValue(context.Background(), DebugContextKey, n)
}

// AddTransition implements [DebuggedCall].
func (n *noop) AddTransition(
	_ string,
	_ CallDirection,
	_ time.Time, _ time.Time,
	_ CallResult,
	_ string) {
}

// Finalise implements [DebuggedCall].
func (n *noop) Finalise() {}

// SetEndTime implements [DebuggedCall].
func (n *noop) SetEndTime(_ time.Time) {}

// SetMethod implements [DebuggedCall].
func (n *noop) SetMethod(_ string) {}

// SetStartTime implements [DebuggedCall].
func (n *noop) SetStartTime(_ time.Time) {}

// SetStatusCode implements [DebuggedCall].
func (n *noop) SetStatusCode(_ int) {}

// SetURL implements [DebuggedCall].
func (n *noop) SetURL(_ string) {}
