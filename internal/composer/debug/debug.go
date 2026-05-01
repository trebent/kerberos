package debug

import "sync/atomic"

// nolint:gochecknoglobals // global provider follows the OTEL provider pattern.
var global = newGlobal()

type globalState struct {
	ptr atomic.Pointer[Debugger]
}

func newGlobal() *globalState {
	g := &globalState{}
	var d Debugger = noopDebuggerSingleton
	g.ptr.Store(&d)
	return g
}

// GetDebugger returns the global Debugger.
// If no debugger has been set, a no-op Debugger is returned.
// GetDebugger is safe for concurrent use and is designed to be called on every
// FlowComponent invocation with minimal overhead.
func GetDebugger() Debugger {
	return *global.ptr.Load()
}

// SetDebugger replaces the global Debugger.
// This should be called once during application initialization before any
// FlowComponent begins handling requests.
// SetDebugger is safe for concurrent use.
func SetDebugger(d Debugger) {
	global.ptr.Store(&d)
}
