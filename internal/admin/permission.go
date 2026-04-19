package admin

import (
	"context"
	"slices"

	"github.com/trebent/kerberos/internal/admin/model"
)

const (
	// Permission IDs — these are stable, fixed values (not auto-incremented).

	PermissionIDFlowViewer          = int64(1)
	PermissionIDOASViewer           = int64(2)
	PermissionIDBasicAuthOrgAdmin   = int64(3)
	PermissionIDBasicAuthOrgViewer  = int64(4)
	PermissionIDAdminUserMgmtAdmin  = int64(5)
	PermissionIDAdminUserMgmtViewer = int64(6)

	// Permission names.

	PermissionNameFlowViewer          = "flowviewer"
	PermissionNameOASViewer           = "oasviewer"
	PermissionNameBasicAuthOrgAdmin   = "basicauthorgadmin"
	PermissionNameBasicAuthOrgViewer  = "basicauthorgviewer"
	PermissionNameAdminUserMgmtAdmin  = "adminusermgmtadmin"
	PermissionNameAdminUserMgmtViewer = "adminusermgmtviewer"
)

// ContextSessionValid reports whether the context contains an admin session.
// An admin session being present means it is valid.
func ContextSessionValid(ctx context.Context) bool {
	val := ctx.Value(adminContextSession)
	_, ok := val.(*model.Session)
	return ok
}

// IsSuperUserContext reports whether the context contains a superuser session.
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
