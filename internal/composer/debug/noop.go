package debug

import "context"

// nolint:gochecknoglobals // singleton instances avoid allocations on the noop hot-path.
var (
	noopDebuggerSingleton = &noopDebugger{}
	noopActionSingleton   = &noopAction{}
)

var (
	_ Debugger = noopDebuggerSingleton
	_ Action   = noopActionSingleton
)

type noopDebugger struct{}

func (n *noopDebugger) StartAction(_ context.Context, _ string) Action {
	return noopActionSingleton
}

type noopAction struct{}

func (n *noopAction) End(_ Outcome) {}
