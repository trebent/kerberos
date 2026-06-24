package debug

import (
	"context"
	"time"
)

type (
	// CallDirection represents the direction of a call, either inbound or outbound.
	CallDirection string
	// CallResult represents the result of a call, either success or failure.
	CallResult string

	// DebuggedCall is the interface that defines the methods for a debugged call.
	DebuggedCall interface {
		// SetStartTime sets the start time of the call.
		SetStartTime(startTime time.Time)
		// SetEndTime sets the end time of the call.
		SetEndTime(endTime time.Time)
		// SetURL sets the URL of the call.
		SetURL(url string)
		// SetMethod sets the HTTP method of the call.
		SetMethod(method string)
		// SetStatusCode sets the HTTP status code of the call.
		SetStatusCode(statusCode int)
		// AddTransition adds a flow transition to the call.
		AddTransition(
			component string,
			direction CallDirection,
			startTime time.Time,
			endTime time.Time,
			result CallResult,
			failureCause string,
		)
		// Finalise finalises the call and makes it ready for storage.
		Finalise()
	}

	// Debugger is the interface that defines the methods for debugging calls.
	Debugger interface {
		// Start starts a new debugged call and returns it along with a context that has the call stored in it.
		Start(context.Context) (DebuggedCall, context.Context)
	}
)

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"

	CallResultSuccess CallResult = "success"
	CallResultFailure CallResult = "failure"

	// DebugContextKey is the context key used to store the debugged call in the context. DO NOT
	// use this key directly, use [composer.DebugContextKey] instead.
	DebugContextKey string = "krb.debug"
)

// SetEndTime sets the end time of a DebuggedCall to the current time. This is a utility
// to be able to defer setting the end time.
func SetEndTime(call DebuggedCall) {
	call.SetEndTime(time.Now())
}
