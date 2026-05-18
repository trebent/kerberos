package debug_test

import (
	"context"
	"testing"
	"time"

	"github.com/trebent/kerberos/internal/composer/debug"
)

func TestGetDebugger_defaultIsNoop(t *testing.T) {
	d := debug.GetDebugger()
	if d == nil {
		t.Fatal("expected non-nil Debugger")
	}

	// The default noop debugger must not panic.
	action := d.StartAction(context.Background(), "test-component")
	if action == nil {
		t.Fatal("expected non-nil Action")
	}

	action.End(debug.Outcome{StatusCode: 200, Duration: time.Millisecond})
}

func TestSetDebugger_replacesGlobal(t *testing.T) {
	original := debug.GetDebugger()
	t.Cleanup(func() { debug.SetDebugger(original) })

	called := false
	debug.SetDebugger(&spyDebugger{onStartAction: func() { called = true }})

	debug.GetDebugger().StartAction(context.Background(), "component").
		End(debug.Outcome{StatusCode: 200})

	if !called {
		t.Fatal("expected custom debugger to be called")
	}
}

func TestSetDebugger_nilNoPanic(t *testing.T) {
	original := debug.GetDebugger()
	t.Cleanup(func() { debug.SetDebugger(original) })

	// Setting a nil Debugger should not panic; retrieving and calling it may
	// panic depending on usage, but the set operation itself must be safe.
	debug.SetDebugger(nil)
}

// spyDebugger is a test helper that records StartAction calls.
type spyDebugger struct {
	onStartAction func()
}

func (s *spyDebugger) StartAction(_ context.Context, _ string) debug.Action {
	if s.onStartAction != nil {
		s.onStartAction()
	}
	return &spyAction{}
}

type spyAction struct{}

func (s *spyAction) End(_ debug.Outcome) {}

// BenchmarkNoopDebugger verifies that the noop path causes zero heap allocations.
func BenchmarkNoopDebugger(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		action := debug.GetDebugger().StartAction(ctx, "test-component")
		action.End(debug.Outcome{StatusCode: 200, Duration: time.Millisecond})
	}
}
