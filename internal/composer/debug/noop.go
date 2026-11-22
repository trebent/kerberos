package debug

import (
	"time"
)

type noopCall struct{}

var noop Call = &noopCall{}

// NoopCall returns a new instance of a DebuggedCall that does not perform any actual debugging.
func NoopCall() Call {
	return noop
}

// AddTransition implements [debug.DebuggedCall].
func (n *noopCall) AddTransition(
	_ string,
	_ Direction,
	_ time.Time, _ time.Time,
	_ Result,
	_ string) {
}

// SetMethod implements [debug.DebuggedCall].
func (n *noopCall) SetMethod(_ string) {}

// SetStartTime implements [debug.DebuggedCall].
func (n *noopCall) SetStartTime(_ time.Time) {}

// SetStatusCode implements [debug.DebuggedCall].
func (n *noopCall) SetStatusCode(_ int) {}

// SetURL implements [debug.DebuggedCall].
func (n *noopCall) SetURL(_ string) {}

func (n *noopCall) Finalise() {}
