package admin

import (
	"context"

	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
)

// ExtendDebugSession implements [withExtensions].
func (i *impl) ExtendDebugSession(
	_ context.Context,
	_ adminapi.ExtendDebugSessionRequestObject,
) (adminapi.ExtendDebugSessionResponseObject, error) {
	panic("unimplemented")
}

// GetDebugSession implements [withExtensions].
func (i *impl) GetDebugSession(
	_ context.Context,
	_ adminapi.GetDebugSessionRequestObject,
) (adminapi.GetDebugSessionResponseObject, error) {
	panic("unimplemented")
}

// ListDebugSessionOperations implements [withExtensions].
func (i *impl) ListDebugSessionOperations(
	_ context.Context,
	_ adminapi.ListDebugSessionOperationsRequestObject,
) (adminapi.ListDebugSessionOperationsResponseObject, error) {
	panic("unimplemented")
}

// ListDebugSessions implements [withExtensions].
func (i *impl) ListDebugSessions(
	_ context.Context,
	_ adminapi.ListDebugSessionsRequestObject,
) (adminapi.ListDebugSessionsResponseObject, error) {
	panic("unimplemented")
}

// StartDebugSession implements [withExtensions].
func (i *impl) StartDebugSession(
	_ context.Context,
	_ adminapi.StartDebugSessionRequestObject,
) (adminapi.StartDebugSessionResponseObject, error) {
	panic("unimplemented")
}

// StopDebugSession implements [withExtensions].
func (i *impl) StopDebugSession(
	_ context.Context,
	_ adminapi.StopDebugSessionRequestObject,
) (adminapi.StopDebugSessionResponseObject, error) {
	panic("unimplemented")
}
