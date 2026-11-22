package admin

import (
	"sync"
	"testing"
	"time"
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
