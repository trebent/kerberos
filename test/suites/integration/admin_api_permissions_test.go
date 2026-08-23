package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

// Permission IDs are the fixed permission IDs bootstrapped by the server.
const (
	PermissionIDFlowViewer          = 1
	PermissionIDOASViewer           = 2
	PermissionIDBasicAuthOrgAdmin   = 3
	PermissionIDBasicAuthOrgViewer  = 4
	PermissionIDAdminUserMgmtAdmin  = 5
	PermissionIDAdminUserMgmtViewer = 6
	PermissionIDDebugger            = 7

	// Permission names.

	PermissionNameFlowViewer          = "flow-viewer"
	PermissionNameOASViewer           = "oas-viewer"
	PermissionNameBasicAuthOrgAdmin   = "basic-auth-org-admin"
	PermissionNameBasicAuthOrgViewer  = "basic-auth-org-viewer"
	PermissionNameAdminUserMgmtAdmin  = "admin-user-mgmt-admin"
	PermissionNameAdminUserMgmtViewer = "admin-user-mgmt-viewer"
	PermissionNameDebugger            = "debugger"
)

// --- GetPermissions ---

// TestAdminPermissionsGetPermissions verifies that any authenticated admin user can list
// available permissions.
func TestAdminPermissionsGetPermissions(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	resp, err := lib.AdminClient.GetPermissionsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusOK, t)

	if resp.JSON200 == nil || len(*resp.JSON200) == 0 {
		t.Fatal("expected non-empty permissions list")
	}

	// Verify the four expected permissions exist with the correct names.
	nameByID := make(map[int]string, len(*resp.JSON200))
	for _, p := range *resp.JSON200 {
		nameByID[p.Id] = p.Name
	}

	expected := map[int]string{
		PermissionIDFlowViewer:          PermissionNameFlowViewer,
		PermissionIDOASViewer:           PermissionNameOASViewer,
		PermissionIDBasicAuthOrgViewer:  PermissionNameBasicAuthOrgViewer,
		PermissionIDBasicAuthOrgAdmin:   PermissionNameBasicAuthOrgAdmin,
		PermissionIDAdminUserMgmtViewer: PermissionNameAdminUserMgmtViewer,
		PermissionIDAdminUserMgmtAdmin:  PermissionNameAdminUserMgmtAdmin,
		PermissionIDDebugger:            PermissionNameDebugger,
	}
	for id, name := range expected {
		if nameByID[id] != name {
			t.Errorf("permission ID %d: expected name %q, got %q", id, name, nameByID[id])
		}
	}
}

// --- Superuser access ---

// TestAdminPermissionsSuperuserAccessAll verifies that the superuser can access every
// permission-gated endpoint without being a member of any group.
func TestAdminPermissionsSuperuserAccessAll(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	// GetFlow — requires flowviewer.
	getFlowResp, err := lib.AdminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getFlowResp.StatusCode(), http.StatusOK, t)

	// GetBackendOAS — requires oasviewer.
	getOASResp, err := lib.AdminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getOASResp.StatusCode(), http.StatusOK, t)

	// Basic auth endpoint (GET) — requires basicauthorgadmin or basicauthorgviewer.
	orgID, _ := lib.OrgWithSession(t, superRequestEditor)
	listUsersResp, err := lib.BasicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// Basic auth endpoint (non-GET) — requires basicauthorgadmin.
	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	// Admin user mgmt (GET) — requires adminusermgmtadmin or adminusermgmtviewer.
	getUsersResp, err := lib.AdminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getUsersResp.StatusCode(), http.StatusOK, t)

	// Admin user mgmt (non-GET) — requires adminusermgmtadmin.
	createUserResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: lib.Username(), Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)

	// Debug (GET) — requires debugger.
	listDebugResp, err := lib.AdminClient.ListDebugSessionsWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listDebugResp.StatusCode(), http.StatusOK, t)

	// Debug (POST) — requires debugger.
	startDebugResp, err := lib.AdminClient.StartDebugSessionWithResponse(
		t.Context(),
		"echo",
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(startDebugResp.StatusCode(), http.StatusOK, t)

	// Clean up the debug session started above.
	deleteDebugResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		startDebugResp.JSON200.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteDebugResp.StatusCode(), http.StatusNoContent, t)
}

// --- flowviewer permission ---

// TestAdminPermissionsFlowViewerAllowed verifies that an admin user with the flowviewer
// permission can call GetFlow.
func TestAdminPermissionsFlowViewerAllowed(t *testing.T) {
	t.Parallel()
	superSession := lib.SuperLogin(t)
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superSession, []int{PermissionIDFlowViewer})

	resp, err := lib.AdminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusOK, t)
}

// TestAdminPermissionsFlowViewerDeniedWithoutPermission verifies that an admin user without
// the flowviewer permission receives 403 when calling GetFlow.
func TestAdminPermissionsFlowViewerDeniedWithoutPermission(t *testing.T) {
	t.Parallel()
	superSession := lib.SuperLogin(t)
	// Give only oasviewer — no flowviewer.
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superSession, []int{PermissionIDOASViewer})

	resp, err := lib.AdminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusForbidden, t)
}

// TestAdminPermissionsFlowViewerDeniedNoGroup verifies that an admin user in no group at all
// receives 403 when calling GetFlow.
func TestAdminPermissionsFlowViewerDeniedNoGroup(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	const pass = "testpassword1"
	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	adminRequestEditor := lib.AdminUserLogin(t, name, pass)

	resp, err := lib.AdminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusForbidden, t)
}

// --- oasviewer permission ---

// TestAdminPermissionsOASViewerAllowed verifies that an admin user with the oasviewer
// permission can call GetBackendOAS.
func TestAdminPermissionsOASViewerAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDOASViewer})

	resp, err := lib.AdminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusOK, t)
}

// TestAdminPermissionsOASViewerDeniedWithoutPermission verifies that an admin user without
// the oasviewer permission receives 403 when calling GetBackendOAS.
func TestAdminPermissionsOASViewerDeniedWithoutPermission(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)
	// Give only flowviewer — no oasviewer.
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDFlowViewer})

	resp, err := lib.AdminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusForbidden, t)
}

// --- basicauthorgadmin permission ---

// TestAdminPermissionsBasicAuthOrgAdminAllowed verifies that an admin user with the
// basicauthorgadmin permission can perform both read and write operations on the
// basic auth API.
func TestAdminPermissionsBasicAuthOrgAdminAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDBasicAuthOrgAdmin})

	// basicauthorgadmin must be able to create an organisation (write).
	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	orgID := createOrgResp.JSON201.Id

	// basicauthorgadmin must be able to list users (read).
	listUsersResp, err := lib.BasicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// basicauthorgadmin must be able to create a user (write).
	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
}

// TestAdminPermissionsBasicAuthOrgAdminDeniedWithoutPermission verifies that an admin user
// without any basic auth permission cannot access the basic auth API. The middleware falls
// through to session lookup (which does not recognise an admin session), returning 401.
func TestAdminPermissionsBasicAuthOrgAdminDeniedWithoutPermission(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)
	// Give only flowviewer — no basic auth permission.
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDFlowViewer})

	// The admin session is not a valid basic auth session, so the middleware returns 401.
	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusUnauthorized, t)
}

// --- basicauthorgviewer permission ---

// TestAdminPermissionsBasicAuthOrgViewerReadAllowed verifies that an admin user with the
// basicauthorgviewer permission can call GET endpoints on the basic auth API.
func TestAdminPermissionsBasicAuthOrgViewerReadAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	// Create an org via the superuser first so there is something to read.
	orgID, _ := lib.OrgWithSession(t, superRequestEditor)

	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDBasicAuthOrgViewer})

	// basicauthorgviewer must be able to list users (GET).
	listUsersResp, err := lib.BasicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// basicauthorgviewer must be able to list groups (GET).
	listGroupsResp, err := lib.BasicAuthClient.ListGroupsWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listGroupsResp.StatusCode(), http.StatusOK, t)
}

// TestAdminPermissionsBasicAuthOrgViewerWriteDenied verifies that an admin user with the
// basicauthorgviewer permission is denied for non-GET (write) endpoints on the basic
// auth API.
func TestAdminPermissionsBasicAuthOrgViewerWriteDenied(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDBasicAuthOrgViewer})

	// basicauthorgviewer must NOT be able to create an organisation (POST).
	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusForbidden, t)

	orgID, _ := lib.OrgWithSession(t, superRequestEditor)
	// Also verify that a user-creation call (POST) is denied.
	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusForbidden, t)
}

// TestAdminPermissionsBasicAuthOrgViewerDeniedWithoutPermission verifies that an admin user
// with no basic auth permission cannot access even GET endpoints on the basic auth API.
// The middleware falls through to session lookup (which does not recognise an admin session),
// returning 401.
func TestAdminPermissionsBasicAuthOrgViewerDeniedWithoutPermission(t *testing.T) {
	t.Parallel()
	superSession := lib.SuperLogin(t)
	orgID, _ := lib.OrgWithSession(t, superSession)

	// Give only flowviewer — no basic auth permission.
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superSession, []int{PermissionIDFlowViewer})

	listUsersResp, err := lib.BasicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listUsersResp.StatusCode(), http.StatusUnauthorized, t)
}

// --- Group response includes permissions ---

// TestAdminPermissionsGroupResponseIncludesPermissions verifies that the permissions field is
// present and accurate in the group create/get responses.
func TestAdminPermissionsGroupResponseIncludesPermissions(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	permIDs := []int{PermissionIDFlowViewer, PermissionIDOASViewer}
	createResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: permIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	if createResp.JSON201.Permissions == nil || len(*createResp.JSON201.Permissions) == 0 {
		t.Fatal("expected permissions in create group response, got empty slice")
	}

	returnedIDs := make([]int, 0, len(*createResp.JSON201.Permissions))
	for _, p := range *createResp.JSON201.Permissions {
		returnedIDs = append(returnedIDs, p.Id)
	}
	lib.ContainsAll(permIDs, returnedIDs, t)
	lib.ContainsAll(returnedIDs, permIDs, t)

	// Verify the same data is returned by GetGroup.
	getResp, err := lib.AdminClient.GetGroupWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)

	if getResp.JSON200.Permissions == nil || len(*getResp.JSON200.Permissions) == 0 {
		t.Fatal("expected permissions in get group response, got empty slice")
	}

	getReturnedIDs := make([]int, 0, len(*getResp.JSON200.Permissions))
	for _, p := range *getResp.JSON200.Permissions {
		getReturnedIDs = append(getReturnedIDs, p.Id)
	}
	lib.ContainsAll(permIDs, getReturnedIDs, t)
	lib.ContainsAll(getReturnedIDs, permIDs, t)
}

// --- adminusermgmtadmin permission ---

// TestAdminPermissionsAdminUserMgmtAdminAllowed verifies that an admin user with the
// adminusermgmtadmin permission can perform both read and write operations on the
// admin user and group management endpoints.
func TestAdminPermissionsAdminUserMgmtAdminAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDAdminUserMgmtAdmin})

	// adminusermgmtadmin must be able to list users (GET).
	listUsersResp, err := lib.AdminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// adminusermgmtadmin must be able to create a user (POST).
	name := lib.Username()
	const pass = "testpassword1"
	createUserResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)

	// adminusermgmtadmin must be able to list users (GET).
	userID := lib.MustGetAdminUserID(t, adminRequestEditor, name)

	// adminusermgmtadmin must be able to get a user (GET).
	getUserResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		userID,
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getUserResp.StatusCode(), http.StatusOK, t)

	// adminusermgmtadmin must be able to list groups (GET).
	listGroupsResp, err := lib.AdminClient.GetGroupsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listGroupsResp.StatusCode(), http.StatusOK, t)

	// adminusermgmtadmin must be able to create a group (POST).
	createGroupResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: []int{}},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)

	groupID := createGroupResp.JSON201.Id

	// adminusermgmtadmin must be able to update user–group bindings (PUT).
	updateGroupsResp, err := lib.AdminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{groupID}},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateGroupsResp.StatusCode(), http.StatusNoContent, t)

	// adminusermgmtadmin must be able to update a group (PUT).
	newGroupName := lib.GroupName()
	updateGroupResp, err := lib.AdminClient.UpdateGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.UpdateGroupJSONRequestBody{Name: newGroupName, PermissionIDs: []int{}},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateGroupResp.StatusCode(), http.StatusNoContent, t)

	// adminusermgmtadmin must be able to delete a user (DELETE).
	deleteUserResp, err := lib.AdminClient.DeleteUserWithResponse(
		t.Context(),
		userID,
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteUserResp.StatusCode(), http.StatusNoContent, t)

	// adminusermgmtadmin must be able to delete a group (DELETE).
	deleteGroupResp, err := lib.AdminClient.DeleteGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteGroupResp.StatusCode(), http.StatusNoContent, t)
}

// --- adminusermgmtviewer permission ---

// TestAdminPermissionsAdminUserMgmtViewerReadAllowed verifies that an admin user with the
// adminusermgmtviewer permission can call GET endpoints on the admin user/group mgmt API.
func TestAdminPermissionsAdminUserMgmtViewerReadAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDAdminUserMgmtViewer})

	// adminusermgmtviewer must be able to list users (GET).
	listUsersResp, err := lib.AdminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// adminusermgmtviewer must be able to list groups (GET).
	listGroupsResp, err := lib.AdminClient.GetGroupsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listGroupsResp.StatusCode(), http.StatusOK, t)
}

// TestAdminPermissionsAdminUserMgmtViewerWriteDenied verifies that an admin user with the
// adminusermgmtviewer permission is denied for non-GET (write) endpoints on the admin
// user/group mgmt API.
func TestAdminPermissionsAdminUserMgmtViewerWriteDenied(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDAdminUserMgmtViewer})

	// adminusermgmtviewer must NOT be able to create a user (POST).
	createUserResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: lib.Username(), Password: "testpassword1"},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusForbidden, t)

	// adminusermgmtviewer must NOT be able to create a group (POST).
	createGroupResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: []int{}},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupResp.StatusCode(), http.StatusForbidden, t)
}

// TestAdminPermissionsAdminUserMgmtViewerDeniedWithoutPermission verifies that an admin user
// with no user mgmt permission receives 403 when calling even GET user mgmt endpoints.
func TestAdminPermissionsAdminUserMgmtViewerDeniedWithoutPermission(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)
	// Give only flowviewer — no user mgmt permission.
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDFlowViewer})

	listUsersResp, err := lib.AdminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listUsersResp.StatusCode(), http.StatusForbidden, t)
}

// TestAdminPermissionsAdminUserMgmtViewerGetSelf verifies that an admin user
// with no user mgmt permission can still get their own user information.
func TestAdminPermissionsAdminUserMgmtViewerGetSelf(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "pass"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	userRequestEditor := lib.AdminUserLogin(t, name, "pass")

	listUsersResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(userRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)
	lib.Matches(listUsersResp.JSON200.Username, name, t)
	lib.Matches(listUsersResp.JSON200.Id, createResp.JSON201.Id, t)
}

// TestAdminPermissionsNormalUserLogoutSuper verifies that a normal admin user, even with permissions to call the logout
// endpoint, cannot log out the superuser.
func TestAdminPermissionsNormalUserLogoutSuper(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDAdminUserMgmtViewer})

	// Normal admin users should not be able to log out the superuser, even if they
	// have permissions to call the logout endpoint.
	logoutResp, err := lib.AdminClient.LogoutSuperuserWithResponse(
		t.Context(), adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(logoutResp.StatusCode(), http.StatusForbidden, t)
}

// TestAdminPermissionsAdminUserChangePasswordWrongUser verifies that an admin user cannot change another user's
// password without the appropriate permission.
func TestAdminPermissionsAdminUserChangePasswordWrongUser(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	const pass = "correctpassword123"

	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	name2 := lib.Username()
	createResp2, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name2, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp2.StatusCode(), http.StatusCreated, t)

	userRequestEditor := lib.AdminUserLogin(t, name2, pass)

	changeResp, err := lib.AdminClient.ChangeUserPasswordWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.ChangeUserPasswordJSONRequestBody{OldPassword: pass, NewPassword: "newpass"},
		adminapi.RequestEditorFn(userRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(changeResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAdminAPIErrorResponse(changeResp.JSON403, t)
}

// --- debugger permission ---

// TestAdminPermissionsDebuggerAllowed verifies that an admin user with the debugger permission
// can call StartDebugSession.
func TestAdminPermissionsDebuggerAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDDebugger})

	resp, err := lib.AdminClient.StartDebugSessionWithResponse(
		t.Context(),
		"echo",
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusOK, t)

	// Clean up.
	deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		resp.JSON200.Id,
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestAdminPermissionsDebuggerDenied verifies that an admin user without the debugger permission
// receives 403 when calling StartDebugSession.
func TestAdminPermissionsDebuggerDenied(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)
	// Give only flowviewer — no debugger.
	adminRequestEditor := lib.CreateAdminUserInGroup(t, superRequestEditor, []int{PermissionIDFlowViewer})

	resp, err := lib.AdminClient.StartDebugSessionWithResponse(
		t.Context(),
		"echo",
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAdminAPIErrorResponse(resp.JSON403, t)
}
