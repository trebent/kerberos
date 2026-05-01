package debug

import (
	"context"
	"time"
)

// Outcome describes the result of a single FlowComponent invocation.
type Outcome struct {
	// StatusCode is the HTTP response status code produced by the component.
	StatusCode int
	// Duration is the time the component took to process the request.
	Duration time.Duration
}

// Action represents a single flow component debug recording session.
// End must be called exactly once after the component finishes processing.
type Action interface {
	// End records the outcome of the flow component invocation.
	End(outcome Outcome)
}

// Debugger records diagnostic information about FlowComponent invocations.
// Implementations must be safe for concurrent use.
type Debugger interface {
	// StartAction begins recording a single flow component invocation.
	// The returned Action must have its End method called when the component finishes.
	StartAction(ctx context.Context, componentName string) Action
}
