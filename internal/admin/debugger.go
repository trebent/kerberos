package admin

import (
	"context"
	"time"

	"github.com/trebent/kerberos/internal/composer"
	composerdebug "github.com/trebent/kerberos/internal/composer/debug"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"
	"golang.org/x/time/rate"
)

type (
	debugger struct {
		db.SQLClient

		*rate.Limiter

		Backends map[string]time.Time
	}
	realCall struct {
		sqlClient db.SQLClient

		url             string
		method          string
		statusCode      int
		startTime       time.Time
		endTime         time.Time
		flowTransitions []flowTransition
	}
	flowTransition struct {
		component    string
		direction    composerdebug.CallDirection
		startTime    time.Time
		endTime      time.Time
		result       composerdebug.CallResult
		failureCause string
	}
)

var (
	_ composerdebug.Debugger     = &debugger{}
	_ composerdebug.DebuggedCall = &realCall{}
)

// newDebugger creates a new debugger that can be used to debug calls.
// The debugger will use the provided SQLClient to store the debugged calls.
func newDebugger(sqlClient db.SQLClient) *debugger {
	return &debugger{
		SQLClient: sqlClient,
		Limiter:   rate.NewLimiter(rate.Every(1*time.Second), 1),
		Backends:  make(map[string]time.Time),
	}
}

func (d *debugger) EnableBackend(backend string, expires time.Time) {
	d.Backends[backend] = expires
}

func (d *debugger) DisableBackend(backend string) {
	delete(d.Backends, backend)
}

func (d *debugger) IsEnabled(backend string) bool {
	expiry, ok := d.Backends[backend]
	if !ok {
		return false
	}

	return time.Now().Before(expiry)
}

// Start implements [debug.Debugger].
func (d *debugger) Start(ctx context.Context) (composerdebug.DebuggedCall, context.Context) {
	if !d.Allow() {
		return composerdebug.NewNoopCall(), ctx
	}

	//nolint:errcheck // the API contract is trusted.
	if !d.IsEnabled(ctx.Value(composer.BackendContextKey).(string)) {
		return composerdebug.NewNoopCall(), ctx
	}

	zerologr.V(20).Info("No active debug sessions found, returning noop debugger")
	return newRealCall(d.SQLClient), ctx
}

func newRealCall(
	sqlClient db.SQLClient,
) *realCall { //nolint:revive // sqlClient will be read in Finalise once persistence is implemented
	return &realCall{
		sqlClient: sqlClient,
		startTime: time.Now().UTC(),
		// Initialised to a set size of 10 to accomodate a typical flow without having
		// to expand the supporting array.
		flowTransitions: make([]flowTransition, 0, 10),
	}
}

// AddTransition implements [debug.DebuggedCall].
func (r *realCall) AddTransition(
	component string,
	direction composerdebug.CallDirection,
	startTime time.Time, endTime time.Time,
	result composerdebug.CallResult,
	failureCause string,
) {
	r.flowTransitions = append(r.flowTransitions, flowTransition{
		component:    component,
		direction:    direction,
		startTime:    startTime,
		endTime:      endTime,
		result:       result,
		failureCause: failureCause,
	})
}

// SetMethod implements [debug.DebuggedCall].
func (r *realCall) SetMethod(method string) {
	r.method = method
}

// SetStartTime implements [debug.DebuggedCall].
func (r *realCall) SetStartTime(startTime time.Time) {
	r.startTime = startTime
}

// SetStatusCode implements [debug.DebuggedCall].
func (r *realCall) SetStatusCode(statusCode int) {
	r.statusCode = statusCode
}

// SetURL implements [debug.DebuggedCall].
func (r *realCall) SetURL(url string) {
	r.url = url
}

// Finalise implements [debug.DebuggedCall].
func (r *realCall) Finalise() {
	r.endTime = time.Now()
}
