package admin

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/trebent/kerberos/internal/composer"
	composerdebug "github.com/trebent/kerberos/internal/composer/debug"
)

// TestDebuggerConcurrentAccess reads and writes the debugger's backend sessions from several
// goroutines at once, as gateway requests and Admin API calls do. Without locking, the Go runtime
// stops the process with "concurrent map read and map write".
func TestDebuggerConcurrentAccess(t *testing.T) {
	d := newDebugger(nil)
	expires := time.Now().Add(time.Minute)

	const iterations = 10000
	var wg sync.WaitGroup

	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := range iterations {
			d.EnableBackend("backend", int64(i), expires)
		}
	}()
	go func() {
		defer wg.Done()
		for range iterations {
			d.DisableBackend("backend")
		}
	}()
	go func() {
		defer wg.Done()
		for range iterations {
			d.IsEnabled("backend")
		}
	}()
	wg.Wait()

	d.DisableBackend("backend")
	d.EnableBackend("backend", 1, expires)
	if id, ok := d.IsEnabled("backend"); !ok || id != 1 {
		t.Fatalf("expected session 1 to be enabled, got id %d, enabled %v", id, ok)
	}
}

// TestDebuggerRestartedSessionGetsNewID checks that starting a new session replaces an earlier
// one for the same backend, so calls are recorded under the new session.
func TestDebuggerRestartedSessionGetsNewID(t *testing.T) {
	d := newDebugger(nil)

	d.EnableBackend("backend", 1, time.Now().Add(-time.Second))
	d.EnableBackend("backend", 2, time.Now().Add(time.Minute))

	if id, ok := d.IsEnabled("backend"); !ok || id != 2 {
		t.Fatalf("expected session 2 to be enabled, got id %d, enabled %v", id, ok)
	}
}

// TestDebuggerRemovesExpiredSession checks that an expired session is removed when it is read.
func TestDebuggerRemovesExpiredSession(t *testing.T) {
	d := newDebugger(nil)

	d.EnableBackend("backend", 1, time.Now().Add(-time.Second))

	if _, ok := d.IsEnabled("backend"); ok {
		t.Fatal("expected an expired session to be disabled")
	}
	if _, ok := d.backendSessions["backend"]; ok {
		t.Fatal("expected the expired session to be removed")
	}
}

// TestDebuggerRateLimitCountsDebuggedCallsOnly checks that calls to a backend that is not being
// debugged do not use up the rate limit of a backend that is.
func TestDebuggerRateLimitCountsDebuggedCallsOnly(t *testing.T) {
	d := newDebugger(nil)
	d.EnableBackend("debugged", 1, time.Now().Add(time.Minute))

	other := context.WithValue(t.Context(), composer.BackendContextKey, "other")
	for range 1000 {
		if call, _ := d.Start(other); call != composerdebug.NewNoopCall() {
			t.Fatal("expected a no-op call for a backend that is not being debugged")
		}
	}

	debugged := context.WithValue(t.Context(), composer.BackendContextKey, "debugged")
	if call, _ := d.Start(debugged); call == composerdebug.NewNoopCall() {
		t.Fatal("expected the debugged backend's call to be recorded")
	}
}
