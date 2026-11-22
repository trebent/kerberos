package debug

import (
	"context"
	"time"
)

type (
	// Direction represents the direction of a call, either inbound or outbound.
	Direction string
	// Result represents the result of a call, either success or failure.
	Result string

	// Call is the interface that defines the methods for a debugged call.
	Call interface {
		// SetStartTime sets the start time of the call.
		SetStartTime(startTime time.Time)
		// SetURL sets the URL of the call.
		SetURL(url string)
		// SetMethod sets the HTTP method of the call.
		SetMethod(method string)
		// SetStatusCode sets the HTTP status code of the call.
		SetStatusCode(statusCode int)
		// AddTransition adds a flow transition to the call.
		AddTransition(
			component string,
			direction Direction,
			startTime time.Time,
			endTime time.Time,
			result Result,
			failureCause string,
		)
		// Finalise finalises the call and makes it ready for storage.
		Finalise()
	}

	// Debugger is the interface that defines the methods for debugging calls.
	Debugger interface {
		// Debug starts a new debugged call and returns it along with a context that has the call stored in it.
		Debug(context.Context) (Call, context.Context)
	}
)

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"

	ResultSuccess Result = "success"
	ResultFailure Result = "failure"

	// DebugContextKey is the context key used to store the debugged call in the context. DO NOT
	// use this key directly, use [composer.DebugContextKey] instead.
	DebugContextKey string = "krb.debug"
)
