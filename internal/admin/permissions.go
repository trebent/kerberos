package admin

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
