package admin

const (
	// Permission IDs — these are stable, fixed values (not auto-incremented).

	PermissionIDFlowViewer         = int64(1)
	PermissionIDOASViewer          = int64(2)
	PermissionIDBasicAuthOrgAdmin  = int64(3)
	PermissionIDBasicAuthOrgViewer = int64(4)

	// Permission names.

	PermissionNameFlowViewer         = "flowviewer"
	PermissionNameOASViewer          = "oasviewer"
	PermissionNameBasicAuthOrgAdmin  = "basicauthorgadmin"
	PermissionNameBasicAuthOrgViewer = "basicauthorgviewer"
)
