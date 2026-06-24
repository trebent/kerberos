package admin

import (
	"context"
	"time"

	"github.com/trebent/kerberos/internal/composer"
	composerdebug "github.com/trebent/kerberos/internal/composer/debug"
	"github.com/trebent/kerberos/internal/db"
	"golang.org/x/time/rate"
)

type (
	debugger struct {
		db.SQLClient

		*rate.Limiter
	}
	realCall struct {
		URL             string
		Method          string
		StatusCode      int
		StartTime       time.Time
		EndTime         time.Time
		FlowTransitions []flowTransition
	}
	flowTransition struct {
		Component    string
		Direction    composerdebug.CallDirection
		StartTime    time.Time
		EndTime      time.Time
		Result       composerdebug.CallResult
		FailureCause string
	}
)

var (
	_ composerdebug.Debugger     = &debugger{}
	_ composerdebug.DebuggedCall = &realCall{}
)

// NewDebugger creates a new debugger that can be used to debug calls.
// The debugger will use the provided SQLClient to store the debugged calls.
func NewDebugger(sqlClient db.SQLClient) composerdebug.Debugger {
	return &debugger{
		SQLClient: sqlClient,
		Limiter:   rate.NewLimiter(rate.Every(1*time.Second), 1),
	}
}

// Start implements [debug.Debugger].
func (d *debugger) Start(ctx context.Context) (composerdebug.DebuggedCall, context.Context) {
	call := noop
	if d.Allow() {
		call = newRealCall()
	}

	return call, context.WithValue(ctx, composer.DebugContextKey, call)
}

func newRealCall() *realCall {
	return &realCall{
		// Initialised to a set size of 10 to accomodate a typical flow without having
		// to expand the supporting array.
		FlowTransitions: make([]flowTransition, 0, 10),
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
	r.FlowTransitions = append(r.FlowTransitions, flowTransition{
		Component:    component,
		Direction:    direction,
		StartTime:    startTime,
		EndTime:      endTime,
		Result:       result,
		FailureCause: failureCause,
	})
}

// SetEndTime implements [debug.DebuggedCall].
func (r *realCall) SetEndTime(endTime time.Time) {
	r.EndTime = endTime
}

// SetMethod implements [debug.DebuggedCall].
func (r *realCall) SetMethod(method string) {
	r.Method = method
}

// SetStartTime implements [debug.DebuggedCall].
func (r *realCall) SetStartTime(startTime time.Time) {
	r.StartTime = startTime
}

// SetStatusCode implements [debug.DebuggedCall].
func (r *realCall) SetStatusCode(statusCode int) {
	r.StatusCode = statusCode
}

// SetURL implements [debug.DebuggedCall].
func (r *realCall) SetURL(url string) {
	r.URL = url
}

// Finalise implements [debug.DebuggedCall].
func (r *realCall) Finalise() {
	panic("unimplemented")
}
