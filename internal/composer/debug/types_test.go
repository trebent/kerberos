package debug

import (
	"context"
	"testing"
	"time"
)

type (
	testDebugger struct {
		returnedCall Call
	}
	testDebugTransition struct {
		component    string
		direction    Direction
		startTime    time.Time
		endTime      time.Time
		result       Result
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
	_ Call = &testDebuggedCall{}
)

// Implements actual logging.
func (d *testDebuggedCall) SetStartTime(startTime time.Time) {
	d.startTime = startTime
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
	direction Direction,
	startTime time.Time,
	endTime time.Time,
	result Result,
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

func (d *testDebugger) Debug(ctx context.Context) (Call, context.Context) {
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
		d := &dummy{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			call, _ := d.Debug(b.Context())
			call.SetStartTime(time.Now())
			call.SetURL("http://example.com")
			call.SetMethod("GET")
			call.SetStatusCode(200)
			call.AddTransition(
				"component",
				DirectionInbound,
				time.Now(),
				time.Now(),
				ResultSuccess,
				"",
			)
		}
	})

	b.Run("With field set", func(b *testing.B) {
		d := &dummy{
			returnedCall: &noopCall{},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			call, _ := d.Debug(b.Context())
			call.SetStartTime(time.Now())
			call.SetURL("http://example.com")
			call.SetMethod("GET")
			call.SetStatusCode(200)
			call.AddTransition(
				"component",
				DirectionInbound,
				time.Now(),
				time.Now(),
				ResultSuccess,
				"",
			)
		}
	})
}
