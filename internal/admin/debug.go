package admin

import (
	"context"

	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
)

// ExtendDebugSession implements [withExtensions].
func (i *impl) ExtendDebugSession(
	ctx context.Context,
	_ adminapi.ExtendDebugSessionRequestObject,
) (adminapi.ExtendDebugSessionResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.ExtendDebugSession403JSONResponse(apiErrForbidden), nil
	}
	panic("unimplemented")
}

// GetDebugSession implements [withExtensions].
func (i *impl) GetDebugSession(
	ctx context.Context,
	_ adminapi.GetDebugSessionRequestObject,
) (adminapi.GetDebugSessionResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.GetDebugSession403JSONResponse(apiErrForbidden), nil
	}
	panic("unimplemented")
}

// ListDebugSessionCalls implements [withExtensions].
func (i *impl) ListDebugSessionCalls(
	ctx context.Context,
	_ adminapi.ListDebugSessionCallsRequestObject,
) (adminapi.ListDebugSessionCallsResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.ListDebugSessionCalls403JSONResponse(apiErrForbidden), nil
	}
	panic("unimplemented")
}

// ListDebugSessions implements [withExtensions].
func (i *impl) ListDebugSessions(
	ctx context.Context,
	_ adminapi.ListDebugSessionsRequestObject,
) (adminapi.ListDebugSessionsResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.ListDebugSessions403JSONResponse(apiErrForbidden), nil
	}
	panic("unimplemented")
}

// StartDebugSession implements [withExtensions].
func (i *impl) StartDebugSession(
	ctx context.Context,
	_ adminapi.StartDebugSessionRequestObject,
) (adminapi.StartDebugSessionResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.StartDebugSession403JSONResponse(apiErrForbidden), nil
	}
	panic("unimplemented")
}

// StopDebugSession implements [withExtensions].
func (i *impl) StopDebugSession(
	ctx context.Context,
	_ adminapi.StopDebugSessionRequestObject,
) (adminapi.StopDebugSessionResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.StopDebugSession403JSONResponse(apiErrForbidden), nil
	}
	panic("unimplemented")
}

// DeleteDebugSession implements [withExtensions].
func (i *impl) DeleteDebugSession(
	ctx context.Context,
	_ adminapi.DeleteDebugSessionRequestObject,
) (adminapi.DeleteDebugSessionResponseObject, error) {
	if !ContextIsDebugger(ctx) {
		return adminapi.DeleteDebugSession403JSONResponse(apiErrForbidden), nil
	}
	panic("unimplemented")
}
