package admin

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/trebent/kerberos/internal/admin/model"
	adminapigen "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/zerologr"
)

type adminContextKey int

const (
	adminContextIsSuperUser adminContextKey = 0
	adminContextSession     adminContextKey = 1
	adminContextPermissions adminContextKey = 2
)

// SessionMiddleware provides context population of administration session information.
// If put in front of a handler, the context to the handler will contain information that can
// be used to determine if a KRB admin made the call, or if it was external.
func SessionMiddleware(
	ssi adminapigen.StrictServerInterface,
) adminapigen.StrictMiddlewareFunc {
	apiImpl, ok := ssi.(*impl)
	if !ok {
		panic("expected admin api *impl")
	}

	return func(
		f nethttp.StrictHTTPHandlerFunc,
		_ string,
	) nethttp.StrictHTTPHandlerFunc {
		return func(
			ctx context.Context,
			w http.ResponseWriter,
			r *http.Request,
			request any,
		) (any, error) {
			zerologr.V(20).Info("Running admin session middleware")

			sessionID := r.Header.Get("X-Krb-Session")

			// No session at all to verify.
			if sessionID == "" {
				return f(ctx, w, r, request)
			}

			session, err := dbGetSession(ctx, apiImpl.sqlClient, sessionID)
			// Not found among sessions, just continue. Remember this middleware does NOT enforce
			// auth, it only populates metadata.
			if err != nil {
				return f(ctx, w, r, request)
			}

			if time.Now().UnixMilli() > session.Expires {
				return f(ctx, w, r, request)
			}

			ctx = context.WithValue(ctx, adminContextSession, session)
			if session.IsSuper {
				ctx = context.WithValue(ctx, adminContextIsSuperUser, true)
			} else {
				// Populate the user's permissions from their group memberships.
				permIDs, err := dbGetUserPermissionIDs(ctx, apiImpl.sqlClient, session.UserID)
				if err != nil {
					zerologr.Error(
						err,
						"Failed to fetch user permissions for session; continuing with no permissions",
						"userID",
						session.UserID,
					)
					// Continue without permissions rather than blocking the request — the endpoint
					// will deny access if a permission is required.
					permIDs = []int64{}
				}
				session.Permissions = permIDs
				ctx = context.WithValue(ctx, adminContextPermissions, permIDs)
			}

			return f(ctx, w, r, request)
		}
	}
}

func RequireSessionMiddleware() adminapigen.StrictMiddlewareFunc {
	return func(
		f nethttp.StrictHTTPHandlerFunc,
		operationID string,
	) nethttp.StrictHTTPHandlerFunc {
		return func(
			ctx context.Context,
			w http.ResponseWriter,
			r *http.Request,
			request any,
		) (any, error) {
			zerologr.V(20).Info("Running admin require session middleware")

			// auto-approve since no session exists.
			if operationID == "LoginSuperuser" || operationID == "Login" {
				return f(ctx, w, r, request)
			}

			if ContextSessionValid(ctx) {
				return f(ctx, w, r, request)
			}

			return nil, apierror.APIErrNoSession
		}
	}
}

func ContextSessionValid(ctx context.Context) bool {
	val := ctx.Value(adminContextSession)
	if session, ok := val.(*model.Session); ok {
		return time.Now().UnixMilli() <= session.Expires
	}

	// No session found in context, invalid.
	return false
}

func IsSuperUserContext(ctx context.Context) bool {
	val := ctx.Value(adminContextIsSuperUser)
	if val == nil {
		return false
	}

	b, ok := val.(bool)
	if !ok {
		return false
	}

	return b
}

// ContextHasPermission reports whether the calling admin user holds the given permission.
// Superusers implicitly hold all permissions.
func ContextHasPermission(ctx context.Context, permissionID int64) bool {
	if IsSuperUserContext(ctx) {
		return true
	}
	val := ctx.Value(adminContextPermissions)
	if val == nil {
		return false
	}

	ids, ok := val.([]int64)
	if !ok {
		return false
	}

	return slices.Contains(ids, permissionID)
}

// ContextCanViewFlow reports whether the calling admin user has the flowviewer permission.
func ContextCanViewFlow(ctx context.Context) bool {
	return ContextHasPermission(ctx, PermissionIDFlowViewer)
}

// ContextCanViewOAS reports whether the calling admin user has the oasviewer permission.
func ContextCanViewOAS(ctx context.Context) bool {
	return ContextHasPermission(ctx, PermissionIDOASViewer)
}

// ContextIsBasicAuthAdmin reports whether the calling admin user has the basicauthorgadmin permission.
func ContextIsBasicAuthAdmin(ctx context.Context) bool {
	return ContextHasPermission(ctx, PermissionIDBasicAuthOrgAdmin)
}

// ContextIsBasicAuthViewer reports whether the calling admin user has the basicauthorgviewer permission.
func ContextIsBasicAuthViewer(ctx context.Context) bool {
	return ContextHasPermission(ctx, PermissionIDBasicAuthOrgViewer)
}

// ContextIsAdminUserMgmtAdmin reports whether the calling admin user has the adminusermgmtadmin permission.
func ContextIsAdminUserMgmtAdmin(ctx context.Context) bool {
	return ContextHasPermission(ctx, PermissionIDAdminUserMgmtAdmin)
}

// ContextIsAdminUserMgmtViewer reports whether the calling admin user has the adminusermgmtviewer permission.
func ContextIsAdminUserMgmtViewer(ctx context.Context) bool {
	return ContextHasPermission(ctx, PermissionIDAdminUserMgmtViewer)
}
