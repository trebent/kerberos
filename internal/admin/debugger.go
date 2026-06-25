package admin

import (
	"context"
	"time"

	"github.com/trebent/kerberos/internal/composer"
	composerdebug "github.com/trebent/kerberos/internal/composer/debug"
	"github.com/trebent/kerberos/internal/db"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	"github.com/trebent/zerologr"
	"golang.org/x/time/rate"
)

type (
	debugger struct {
		db.SQLClient

		*rate.Limiter

		backendSessions map[string]session
	}
	session struct {
		id      int64
		expires time.Time
	}
	realCall struct {
		sqlClient db.SQLClient

		sessionID int64
		apiCall   adminapi.DebugSessionCall
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
		SQLClient:       sqlClient,
		Limiter:         rate.NewLimiter(rate.Every(1*time.Second), 100),
		backendSessions: make(map[string]session),
	}
}

// EnableBackend enables debugging for the specified backend with the given session ID and expiration time.
func (d *debugger) EnableBackend(backend string, id int64, expires time.Time) {
	s, ok := d.backendSessions[backend]
	if ok {
		s.expires = expires
		d.backendSessions[backend] = s
	} else {
		d.backendSessions[backend] = session{
			id:      id,
			expires: expires,
		}
	}
}

// DisableBackend disables debugging for the specified backend.
func (d *debugger) DisableBackend(backend string) {
	delete(d.backendSessions, backend)
}

// IsEnabled checks if debugging is enabled for the specified backend and if the session has not expired.
func (d *debugger) IsEnabled(backend string) (int64, bool) {
	session, ok := d.backendSessions[backend]
	if !ok {
		return 0, false
	}

	return session.id, time.Now().Before(session.expires)
}

// Start implements [debug.Debugger].
func (d *debugger) Start(ctx context.Context) (composerdebug.DebuggedCall, context.Context) {
	if !d.Allow() {
		return composerdebug.NewNoopCall(), ctx
	}

	//nolint:errcheck // the API contract is trusted.
	backend := ctx.Value(composer.BackendContextKey).(string)
	id, enabled := d.IsEnabled(backend)
	if !enabled {
		zerologr.V(20).Info("Backend is not being debugged, returning noop debugger")
		return composerdebug.NewNoopCall(), ctx
	}

	zerologr.V(20).Info("Debugging call", "backend", backend, "session_id", id)
	rc := newRealCall(d.SQLClient, id)
	return rc, context.WithValue(ctx, composer.DebugContextKey, rc)
}

func newRealCall(
	sqlClient db.SQLClient,
	sessionID int64,
) *realCall { //nolint:revive // sqlClient will be read in Finalise once persistence is implemented
	return &realCall{
		sqlClient: sqlClient,
		sessionID: sessionID,
		apiCall: adminapi.DebugSessionCall{
			StartedAt:       time.Now(),
			FlowTransitions: make([]adminapi.FlowTransition, 0, 10),
		},
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
	r.apiCall.FlowTransitions = append(r.apiCall.FlowTransitions, adminapi.FlowTransition{
		Component: component,
		Direction: adminapi.FlowTransitionDirection(direction),
		StartedAt: startTime,
		StoppedAt: endTime,
		Result: adminapi.FlowTransitionResult{
			Outcome: adminapi.FlowTransitionResultOutcome(result),
			Cause:   new(failureCause),
		},
	})
}

// SetMethod implements [debug.DebuggedCall].
func (r *realCall) SetMethod(method string) {
	r.apiCall.Method = method
}

// SetStartTime implements [debug.DebuggedCall].
func (r *realCall) SetStartTime(startTime time.Time) {
	r.apiCall.StartedAt = startTime
}

// SetStatusCode implements [debug.DebuggedCall].
func (r *realCall) SetStatusCode(statusCode int) {
	r.apiCall.StatusCode = statusCode
}

// SetURL implements [debug.DebuggedCall].
func (r *realCall) SetURL(url string) {
	r.apiCall.Url = url
}

// Finalise implements [debug.DebuggedCall].
func (r *realCall) Finalise() {
	r.apiCall.StoppedAt = time.Now()

	_, err := dbCreateDebugSessionCall(context.Background(), r.sqlClient, r.sessionID, r.apiCall)
	if err != nil {
		zerologr.Error(err, "Failed to persist debug session call")
	}
}
