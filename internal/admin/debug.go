package admin

import (
	"context"
	"errors"
	"time"

	admindb "github.com/trebent/kerberos/internal/admin/db"
	"github.com/trebent/kerberos/internal/db"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	"github.com/trebent/zerologr"
)

// StartDebugSession implements [withExtensions].
func (i *impl) StartDebugSession(
	ctx context.Context,
	req adminapi.StartDebugSessionRequestObject,
) (adminapi.StartDebugSessionResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.StartDebugSession403JSONResponse(apiErrForbidden), nil
	}

	sessions, err := admindb.ListDebugSessions(ctx, i.sqlClient, req.Backend)
	if err != nil {
		return adminapi.StartDebugSession500JSONResponse(apiErrInternal), err
	}

	// Keep it simple, only allow 1 debug session per backend.
	if len(sessions) > 0 {
		for _, s := range sessions {
			// Need this check since expired/stopped sessions should be ignored.
			// Check: not expired AND not stopped
			if time.Now().Before(s.ExpiresAt) && s.StoppedAt == nil {
				return adminapi.StartDebugSession409JSONResponse(apiErrConflict), nil
			}
		}
	}

	expires := time.Now().Add(5 * time.Minute).UTC()
	if req.Body != nil && req.Body.DurationSeconds != nil {
		expires = time.Now().Add(time.Duration(*req.Body.DurationSeconds) * time.Second).UTC()
	}

	id, err := admindb.CreateDebugSession(ctx, i.sqlClient, req.Backend, expires)
	if err != nil {
		return adminapi.StartDebugSession500JSONResponse(apiErrInternal), err
	}

	session, err := admindb.GetDebugSession(ctx, i.sqlClient, req.Backend, id)
	if err != nil {
		return adminapi.StartDebugSession500JSONResponse(apiErrInternal), err
	}

	zerologr.Info("Started debug session", "id", id, "backend", req.Backend, "expires", expires)

	i.debugger.EnableBackend(req.Backend, id, session.ExpiresAt)
	return adminapi.StartDebugSession200JSONResponse{
		Id:        int(id),
		Backend:   req.Backend,
		StartedAt: session.StartedAt,
		ExpiresAt: expires,
		StoppedAt: session.StoppedAt,
	}, nil
}

// StopDebugSession implements [withExtensions].
func (i *impl) StopDebugSession(
	ctx context.Context,
	req adminapi.StopDebugSessionRequestObject,
) (adminapi.StopDebugSessionResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.StopDebugSession403JSONResponse(apiErrForbidden), nil
	}

	updatedSession, err := admindb.GetDebugSession(
		ctx,
		i.sqlClient,
		req.Backend,
		int64(req.SessionId),
	)
	if err != nil {
		if errors.Is(err, db.ErrRowNotFound) {
			return adminapi.StopDebugSession404JSONResponse(apiErrNotFound), nil
		}

		return adminapi.StopDebugSession500JSONResponse(apiErrInternal), err
	}

	stoppedAt := time.Now().UTC()
	updatedSession.StoppedAt = &stoppedAt

	if err := admindb.UpdateDebugSession(ctx, i.sqlClient, *updatedSession); err != nil {
		return adminapi.StopDebugSession500JSONResponse(apiErrInternal), err
	}

	zerologr.Info(
		"Stopped debug session",
		"id", req.SessionId,
		"backend", req.Backend,
		"stoppedAt", updatedSession.StoppedAt,
	)

	i.debugger.DisableBackend(req.Backend)
	return adminapi.StopDebugSession204Response{}, nil
}

// ExtendDebugSession implements [withExtensions].
func (i *impl) ExtendDebugSession(
	ctx context.Context,
	req adminapi.ExtendDebugSessionRequestObject,
) (adminapi.ExtendDebugSessionResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.ExtendDebugSession403JSONResponse(apiErrForbidden), nil
	}

	updatedSession, err := admindb.GetDebugSession(
		ctx,
		i.sqlClient,
		req.Backend,
		int64(req.SessionId),
	)
	if err != nil {
		if errors.Is(err, db.ErrRowNotFound) {
			return adminapi.ExtendDebugSession404JSONResponse(apiErrNotFound), nil
		}

		return adminapi.ExtendDebugSession500JSONResponse(apiErrInternal), err
	}

	updatedSession.ExpiresAt = updatedSession.ExpiresAt.Add(
		time.Duration(req.Body.AdditionalDurationSeconds) * time.Second,
	)

	// Verify the expiry isn't more than 1 hour into the future.
	if time.Until(updatedSession.ExpiresAt) > time.Hour {
		updatedSession.ExpiresAt = updatedSession.StartedAt.Add(1 * time.Hour)
	}

	if err := admindb.UpdateDebugSession(ctx, i.sqlClient, *updatedSession); err != nil {
		return adminapi.ExtendDebugSession500JSONResponse(apiErrInternal), err
	}

	zerologr.Info(
		"Extended debug session",
		"id", req.SessionId,
		"backend", req.Backend,
		"expiresAt", updatedSession.ExpiresAt,
	)

	i.debugger.EnableBackend(req.Backend, int64(updatedSession.Id), updatedSession.ExpiresAt)
	return adminapi.ExtendDebugSession200JSONResponse{
		Id:        updatedSession.Id,
		Backend:   updatedSession.Backend,
		StartedAt: updatedSession.StartedAt,
		ExpiresAt: updatedSession.ExpiresAt,
		StoppedAt: updatedSession.StoppedAt,
	}, nil
}

// GetDebugSession implements [withExtensions].
func (i *impl) GetDebugSession(
	ctx context.Context,
	req adminapi.GetDebugSessionRequestObject,
) (adminapi.GetDebugSessionResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.GetDebugSession403JSONResponse(apiErrForbidden), nil
	}

	session, err := admindb.GetDebugSession(ctx, i.sqlClient, req.Backend, int64(req.SessionId))
	if err != nil {
		if errors.Is(err, db.ErrRowNotFound) {
			return adminapi.GetDebugSession404JSONResponse(apiErrNotFound), nil
		}

		return adminapi.GetDebugSession500JSONResponse(apiErrInternal), err
	}

	return adminapi.GetDebugSession200JSONResponse{
		Id:        session.Id,
		Backend:   session.Backend,
		StartedAt: session.StartedAt,
		ExpiresAt: session.ExpiresAt,
		StoppedAt: session.StoppedAt,
	}, nil
}

// ListDebugSessions implements [withExtensions].
func (i *impl) ListDebugSessions(
	ctx context.Context,
	req adminapi.ListDebugSessionsRequestObject,
) (adminapi.ListDebugSessionsResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.ListDebugSessions403JSONResponse(apiErrForbidden), nil
	}

	sessions, err := admindb.ListDebugSessions(ctx, i.sqlClient, req.Backend)
	if err != nil {
		return adminapi.ListDebugSessions500JSONResponse(apiErrInternal), err
	}

	return adminapi.ListDebugSessions200JSONResponse(sessions), nil
}

// DeleteDebugSession implements [withExtensions].
func (i *impl) DeleteDebugSession(
	ctx context.Context,
	req adminapi.DeleteDebugSessionRequestObject,
) (adminapi.DeleteDebugSessionResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.DeleteDebugSession403JSONResponse(apiErrForbidden), nil
	}

	if _, err := admindb.GetDebugSession(
		ctx,
		i.sqlClient,
		req.Backend,
		int64(req.SessionId),
	); err != nil {
		if errors.Is(err, db.ErrRowNotFound) {
			return adminapi.DeleteDebugSession404JSONResponse(apiErrNotFound), nil
		}

		return adminapi.DeleteDebugSession500JSONResponse(apiErrInternal), err
	}

	if err := admindb.DeleteDebugSession(
		ctx, i.sqlClient, req.Backend, int64(req.SessionId),
	); err != nil {
		return adminapi.DeleteDebugSession500JSONResponse(apiErrInternal), err
	}

	zerologr.Info(
		"Deleted debug session",
		"id", req.SessionId,
		"backend", req.Backend,
	)

	i.debugger.DisableBackend(req.Backend)
	return adminapi.DeleteDebugSession204Response{}, nil
}

// ListDebugSessionCalls implements [withExtensions].
func (i *impl) ListDebugSessionCalls(
	ctx context.Context,
	req adminapi.ListDebugSessionCallsRequestObject,
) (adminapi.ListDebugSessionCallsResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.ListDebugSessionCalls403JSONResponse(apiErrForbidden), nil
	}

	calls, err := admindb.ListDebugSessionCalls(
		ctx,
		i.sqlClient,
		int64(req.SessionId),
		req.Params.IncludeTransitions,
	)
	if err != nil {
		return adminapi.ListDebugSessionCalls500JSONResponse(apiErrInternal), err
	}

	return adminapi.ListDebugSessionCalls200JSONResponse(calls), nil
}

func (i *impl) GetDebugSessionCall(
	ctx context.Context,
	req adminapi.GetDebugSessionCallRequestObject,
) (adminapi.GetDebugSessionCallResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.GetDebugSessionCall403JSONResponse(apiErrForbidden), nil
	}

	call, err := admindb.GetDebugSessionCall(
		ctx,
		i.sqlClient,
		int64(req.CallId),
	)
	if err != nil {
		if errors.Is(err, db.ErrRowNotFound) {
			return adminapi.GetDebugSessionCall404JSONResponse(apiErrNotFound), nil
		}

		return adminapi.GetDebugSessionCall500JSONResponse(apiErrInternal), err
	}

	return adminapi.GetDebugSessionCall200JSONResponse(*call), nil
}
