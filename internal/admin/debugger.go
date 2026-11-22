package admin

import (
	"context"
	"sync"
	"time"

	"github.com/trebent/kerberos/internal/composer"
	composerdebug "github.com/trebent/kerberos/internal/composer/debug"
	"github.com/trebent/kerberos/internal/db"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	"github.com/trebent/zerologr"
	"golang.org/x/time/rate"

	admindb "github.com/trebent/kerberos/internal/admin/db"
)

type (
	debugger struct {
		db.SQLClient

		*rate.Limiter

		// mu guards backendSessions, which gateway requests read while Admin API calls write it.
		mu              sync.RWMutex
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
	_ composerdebug.Call = &realCall{}
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
// It replaces any earlier session for the backend, so calls are recorded under the given session.
func (d *debugger) EnableBackend(backend string, id int64, expires time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.backendSessions[backend] = session{
		id:      id,
		expires: expires,
	}
}

// DisableBackend disables debugging for the specified backend.
func (d *debugger) DisableBackend(backend string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.backendSessions, backend)
}

// IsEnabled checks if debugging is enabled for the specified backend and if the session has not expired.
// An expired session is removed.
func (d *debugger) IsEnabled(backend string) (int64, bool) {
	d.mu.RLock()
	s, ok := d.backendSessions[backend]
	d.mu.RUnlock()

	if !ok {
		return 0, false
	}

	if time.Now().Before(s.expires) {
		return s.id, true
	}

	d.mu.Lock()
	// Only remove the session that was found expired: it may have been extended or replaced since.
	if current, ok := d.backendSessions[backend]; ok && current == s {
		delete(d.backendSessions, backend)
	}
	d.mu.Unlock()

	return 0, false
}

// Debug implements [debug.Debugger]. Only calls to a backend being debugged count towards the rate
// limit.
func (d *debugger) Debug(ctx context.Context) (composerdebug.Call, context.Context) {
	//nolint:errcheck // the API contract is trusted.
	backend := ctx.Value(composer.BackendContextKey).(string)
	id, enabled := d.IsEnabled(backend)
	if !enabled || !d.Allow() {
		zerologr.V(20).Info("Call is not debugged")
		noop := composerdebug.NoopCall()
		return noop, context.WithValue(ctx, composer.DebugContextKey, noop)
	}

	zerologr.V(20).Info("Debugging call", "backend", backend, "session_id", id)
	rc := newRealCall(d.SQLClient, id)
	return rc, context.WithValue(ctx, composer.DebugContextKey, rc)
}

func newRealCall(
	sqlClient db.SQLClient,
	sessionID int64,
) composerdebug.Call { //nolint:revive // sqlClient will be read in Finalise once persistence is implemented
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
	direction composerdebug.Direction,
	startTime time.Time, endTime time.Time,
	result composerdebug.Result,
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

	_, err := admindb.CreateDebugSessionCall(
		context.Background(),
		r.sqlClient,
		r.sessionID,
		r.apiCall,
	)
	if err != nil {
		zerologr.Error(err, "Failed to persist debug session call")
	}
}
