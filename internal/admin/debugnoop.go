package admin

import (
	"time"

	composerdebug "github.com/trebent/kerberos/internal/composer/debug"
)

type noopCall struct{}

var noop composerdebug.DebuggedCall = &noopCall{}

// AddTransition implements [debug.DebuggedCall].
func (n *noopCall) AddTransition(
	_ string,
	_ composerdebug.CallDirection,
	_ time.Time, _ time.Time,
	_ composerdebug.CallResult,
	_ string) {
}

// SetEndTime implements [debug.DebuggedCall].
func (n *noopCall) SetEndTime(_ time.Time) {}

// SetMethod implements [debug.DebuggedCall].
func (n *noopCall) SetMethod(_ string) {}

// SetStartTime implements [debug.DebuggedCall].
func (n *noopCall) SetStartTime(_ time.Time) {}

// SetStatusCode implements [debug.DebuggedCall].
func (n *noopCall) SetStatusCode(_ int) {}

// SetURL implements [debug.DebuggedCall].
func (n *noopCall) SetURL(_ string) {}

func (n *noopCall) Finalise() {}
