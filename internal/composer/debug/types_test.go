package debug

import (
	"context"
	"testing"
	"time"
)

type (
	testDebugger struct {
		returnedCall DebuggedCall
	}
	noopCall            struct{}
	testDebugTransition struct {
		component    string
		direction    CallDirection
		startTime    time.Time
		endTime      time.Time
		result       CallResult
		failureCause string
	}
	testDebuggedCall struct {
		startTime   time.Time
		endTime     time.Time
		url         string
		method      string
		statusCode  int
		transitions []testDebugTransition
	}
)

var (
	_ Debugger     = &testDebugger{}
	_ DebuggedCall = &testDebuggedCall{}
)

// all no-ops
func (d *noopCall) SetStartTime(startTime time.Time) {}
func (d *noopCall) SetEndTime(endTime time.Time)     {}
func (d *noopCall) SetURL(url string)                {}
func (d *noopCall) SetMethod(method string)          {}
func (d *noopCall) SetStatusCode(statusCode int)     {}
func (d *noopCall) AddTransition(
	component string,
	direction CallDirection,
	startTime time.Time,
	endTime time.Time,
	result CallResult,
	failureCause string,
) {
}
func (d *noopCall) Finalise() {}

// Implements actuall logging.
func (d *testDebuggedCall) SetStartTime(startTime time.Time) {
	d.startTime = startTime
}

func (d *testDebuggedCall) SetEndTime(endTime time.Time) {
	d.endTime = endTime
}

func (d *testDebuggedCall) SetURL(url string) {
	d.url = url
}

func (d *testDebuggedCall) SetMethod(method string) {
	d.method = method
}

func (d *testDebuggedCall) SetStatusCode(statusCode int) {
	d.statusCode = statusCode
}

func (d *testDebuggedCall) AddTransition(
	component string,
	direction CallDirection,
	startTime time.Time,
	endTime time.Time,
	result CallResult,
	failureCause string,
) {
	d.transitions = append(d.transitions, testDebugTransition{
		component:    component,
		direction:    direction,
		startTime:    startTime,
		endTime:      endTime,
		result:       result,
		failureCause: failureCause,
	})
}

func (d *testDebugger) Start(ctx context.Context) (DebuggedCall, context.Context) {
	call := d.returnedCall
	if call == nil {
		call = &testDebuggedCall{}
	}

	return call, context.WithValue(ctx, DebugContextKey, call)
}

func (d *testDebuggedCall) Finalise() {
}

func BenchmarkCallStart(b *testing.B) {
	b.Run("Without field set", func(b *testing.B) {
		d := &testDebugger{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			call, _ := d.Start(b.Context())
			call.SetStartTime(time.Now())
			call.SetEndTime(time.Now())
			call.SetURL("http://example.com")
			call.SetMethod("GET")
			call.SetStatusCode(200)
			call.AddTransition(
				"component",
				CallDirectionInbound,
				time.Now(),
				time.Now(),
				CallResultSuccess,
				"",
			)
		}
	})

	b.Run("With field set", func(b *testing.B) {
		d := &testDebugger{
			returnedCall: &noopCall{},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			call, _ := d.Start(b.Context())
			call.SetStartTime(time.Now())
			call.SetEndTime(time.Now())
			call.SetURL("http://example.com")
			call.SetMethod("GET")
			call.SetStatusCode(200)
			call.AddTransition(
				"component",
				CallDirectionInbound,
				time.Now(),
				time.Now(),
				CallResultSuccess,
				"",
			)
		}
	})
}
