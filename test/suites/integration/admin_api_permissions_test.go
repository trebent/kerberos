package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/integration/client/auth/basic"
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

// TestPermissionsGetPermissions verifies that any authenticated admin user can list
// available permissions.
func TestPermissionsGetPermissions(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)

	resp, err := adminClient.GetPermissionsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusOK, t)

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

// TestPermissionsSuperuserAccessAll verifies that the superuser can access every
// permission-gated endpoint without being a member of any group.
func TestPermissionsSuperuserAccessAll(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)

	// GetFlow — requires flowviewer.
	getFlowResp, err := adminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getFlowResp.StatusCode(), http.StatusOK, t)

	// GetBackendOAS — requires oasviewer.
	getOASResp, err := adminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getOASResp.StatusCode(), http.StatusOK, t)

	// Basic auth endpoint (GET) — requires basicauthorgadmin or basicauthorgviewer.
	orgID, _ := orgWithSession(t, superRequestEditor)
	listUsersResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// Basic auth endpoint (non-GET) — requires basicauthorgadmin.
	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	// Admin user mgmt (GET) — requires adminusermgmtadmin or adminusermgmtviewer.
	getUsersResp, err := adminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getUsersResp.StatusCode(), http.StatusOK, t)

	// Admin user mgmt (non-GET) — requires adminusermgmtadmin.
	createUserResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: username(), Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)

	// Debug (GET) — requires debugger.
	listDebugResp, err := adminClient.ListDebugSessionsWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listDebugResp.StatusCode(), http.StatusOK, t)

	// Debug (POST) — requires debugger.
	startDebugResp, err := adminClient.StartDebugSessionWithResponse(
		t.Context(),
		"echo",
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(startDebugResp.StatusCode(), http.StatusOK, t)

	// Clean up the debug session started above.
	deleteDebugResp, err := adminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		startDebugResp.JSON200.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(deleteDebugResp.StatusCode(), http.StatusNoContent, t)
}

// --- flowviewer permission ---

// TestPermissionsFlowViewerAllowed verifies that an admin user with the flowviewer
// permission can call GetFlow.
func TestPermissionsFlowViewerAllowed(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)
	adminRequestEditor := createAdminUserInGroup(t, superSession, []int{PermissionIDFlowViewer})

	resp, err := adminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusOK, t)
}

// TestPermissionsFlowViewerDeniedWithoutPermission verifies that an admin user without
// the flowviewer permission receives 403 when calling GetFlow.
func TestPermissionsFlowViewerDeniedWithoutPermission(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)
	// Give only oasviewer — no flowviewer.
	adminRequestEditor := createAdminUserInGroup(t, superSession, []int{PermissionIDOASViewer})

	resp, err := adminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusForbidden, t)
}

// TestPermissionsFlowViewerDeniedNoGroup verifies that an admin user in no group at all
// receives 403 when calling GetFlow.
func TestPermissionsFlowViewerDeniedNoGroup(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)

	const pass = "testpassword1"
	name := username()
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	adminRequestEditor := adminUserLogin(t, name, pass)

	resp, err := adminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusForbidden, t)
}

// --- oasviewer permission ---

// TestPermissionsOASViewerAllowed verifies that an admin user with the oasviewer
// permission can call GetBackendOAS.
func TestPermissionsOASViewerAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDOASViewer})

	resp, err := adminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusOK, t)
}

// TestPermissionsOASViewerDeniedWithoutPermission verifies that an admin user without
// the oasviewer permission receives 403 when calling GetBackendOAS.
func TestPermissionsOASViewerDeniedWithoutPermission(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)
	// Give only flowviewer — no oasviewer.
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDFlowViewer})

	resp, err := adminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusForbidden, t)
}

// --- basicauthorgadmin permission ---

// TestPermissionsBasicAuthOrgAdminAllowed verifies that an admin user with the
// basicauthorgadmin permission can perform both read and write operations on the
// basic auth API.
func TestPermissionsBasicAuthOrgAdminAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDBasicAuthOrgAdmin})

	// basicauthorgadmin must be able to create an organisation (write).
	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	orgID := createOrgResp.JSON201.Id

	// basicauthorgadmin must be able to list users (read).
	listUsersResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// basicauthorgadmin must be able to create a user (write).
	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
}

// TestPermissionsBasicAuthOrgAdminDeniedWithoutPermission verifies that an admin user
// without any basic auth permission cannot access the basic auth API. The middleware falls
// through to session lookup (which does not recognise an admin session), returning 401.
func TestPermissionsBasicAuthOrgAdminDeniedWithoutPermission(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)
	// Give only flowviewer — no basic auth permission.
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDFlowViewer})

	// The admin session is not a valid basic auth session, so the middleware returns 401.
	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusUnauthorized, t)
}

// --- basicauthorgviewer permission ---

// TestPermissionsBasicAuthOrgViewerReadAllowed verifies that an admin user with the
// basicauthorgviewer permission can call GET endpoints on the basic auth API.
func TestPermissionsBasicAuthOrgViewerReadAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)

	// Create an org via the superuser first so there is something to read.
	orgID, _ := orgWithSession(t, superRequestEditor)

	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDBasicAuthOrgViewer})

	// basicauthorgviewer must be able to list users (GET).
	listUsersResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// basicauthorgviewer must be able to list groups (GET).
	listGroupsResp, err := basicAuthClient.ListGroupsWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listGroupsResp.StatusCode(), http.StatusOK, t)
}

// TestPermissionsBasicAuthOrgViewerWriteDenied verifies that an admin user with the
// basicauthorgviewer permission is denied for non-GET (write) endpoints on the basic
// auth API.
func TestPermissionsBasicAuthOrgViewerWriteDenied(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDBasicAuthOrgViewer})

	// basicauthorgviewer must NOT be able to create an organisation (POST).
	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusForbidden, t)

	orgID, _ := orgWithSession(t, superRequestEditor)
	// Also verify that a user-creation call (POST) is denied.
	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusForbidden, t)
}

// TestPermissionsBasicAuthOrgViewerDeniedWithoutPermission verifies that an admin user
// with no basic auth permission cannot access even GET endpoints on the basic auth API.
// The middleware falls through to session lookup (which does not recognise an admin session),
// returning 401.
func TestPermissionsBasicAuthOrgViewerDeniedWithoutPermission(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)
	orgID, _ := orgWithSession(t, superSession)

	// Give only flowviewer — no basic auth permission.
	adminRequestEditor := createAdminUserInGroup(t, superSession, []int{PermissionIDFlowViewer})

	listUsersResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusUnauthorized, t)
}

// --- Group response includes permissions ---

// TestPermissionsGroupResponseIncludesPermissions verifies that the permissions field is
// present and accurate in the group create/get responses.
func TestPermissionsGroupResponseIncludesPermissions(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)

	permIDs := []int{PermissionIDFlowViewer, PermissionIDOASViewer}
	createResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: permIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	if len(createResp.JSON201.Permissions) == 0 {
		t.Fatal("expected permissions in create group response, got empty slice")
	}

	returnedIDs := make([]int, 0, len(createResp.JSON201.Permissions))
	for _, p := range createResp.JSON201.Permissions {
		returnedIDs = append(returnedIDs, p.Id)
	}
	containsAll(permIDs, returnedIDs, t)
	containsAll(returnedIDs, permIDs, t)

	// Verify the same data is returned by GetGroup.
	getResp, err := adminClient.GetGroupWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)

	if len(getResp.JSON200.Permissions) == 0 {
		t.Fatal("expected permissions in get group response, got empty slice")
	}

	getReturnedIDs := make([]int, 0, len(getResp.JSON200.Permissions))
	for _, p := range getResp.JSON200.Permissions {
		getReturnedIDs = append(getReturnedIDs, p.Id)
	}
	containsAll(permIDs, getReturnedIDs, t)
	containsAll(getReturnedIDs, permIDs, t)
}

// --- adminusermgmtadmin permission ---

// TestPermissionsAdminUserMgmtAdminAllowed verifies that an admin user with the
// adminusermgmtadmin permission can perform both read and write operations on the
// admin user and group management endpoints.
func TestPermissionsAdminUserMgmtAdminAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDAdminUserMgmtAdmin})

	// adminusermgmtadmin must be able to list users (GET).
	listUsersResp, err := adminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// adminusermgmtadmin must be able to create a user (POST).
	name := username()
	const pass = "testpassword1"
	createUserResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)

	// adminusermgmtadmin must be able to list users (GET).
	userID := mustGetAdminUserID(t, adminRequestEditor, name)

	// adminusermgmtadmin must be able to get a user (GET).
	getUserResp, err := adminClient.GetUserWithResponse(
		t.Context(),
		userID,
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getUserResp.StatusCode(), http.StatusOK, t)

	// adminusermgmtadmin must be able to list groups (GET).
	listGroupsResp, err := adminClient.GetGroupsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listGroupsResp.StatusCode(), http.StatusOK, t)

	// adminusermgmtadmin must be able to create a group (POST).
	createGroupResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: []int{}},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)

	groupID := createGroupResp.JSON201.Id

	// adminusermgmtadmin must be able to update user–group bindings (PUT).
	updateGroupsResp, err := adminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{groupID}},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(updateGroupsResp.StatusCode(), http.StatusNoContent, t)

	// adminusermgmtadmin must be able to update a group (PUT).
	newGroupName := groupName()
	updateGroupResp, err := adminClient.UpdateGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.UpdateGroupJSONRequestBody{Name: newGroupName, PermissionIDs: []int{}},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(updateGroupResp.StatusCode(), http.StatusNoContent, t)

	// adminusermgmtadmin must be able to delete a user (DELETE).
	deleteUserResp, err := adminClient.DeleteUserWithResponse(
		t.Context(),
		userID,
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(deleteUserResp.StatusCode(), http.StatusNoContent, t)

	// adminusermgmtadmin must be able to delete a group (DELETE).
	deleteGroupResp, err := adminClient.DeleteGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(deleteGroupResp.StatusCode(), http.StatusNoContent, t)
}

// --- adminusermgmtviewer permission ---

// TestPermissionsAdminUserMgmtViewerReadAllowed verifies that an admin user with the
// adminusermgmtviewer permission can call GET endpoints on the admin user/group mgmt API.
func TestPermissionsAdminUserMgmtViewerReadAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDAdminUserMgmtViewer})

	// adminusermgmtviewer must be able to list users (GET).
	listUsersResp, err := adminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// adminusermgmtviewer must be able to list groups (GET).
	listGroupsResp, err := adminClient.GetGroupsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listGroupsResp.StatusCode(), http.StatusOK, t)
}

// TestPermissionsAdminUserMgmtViewerWriteDenied verifies that an admin user with the
// adminusermgmtviewer permission is denied for non-GET (write) endpoints on the admin
// user/group mgmt API.
func TestPermissionsAdminUserMgmtViewerWriteDenied(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDAdminUserMgmtViewer})

	// adminusermgmtviewer must NOT be able to create a user (POST).
	createUserResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: username(), Password: "testpassword1"},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusForbidden, t)

	// adminusermgmtviewer must NOT be able to create a group (POST).
	createGroupResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: []int{}},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupResp.StatusCode(), http.StatusForbidden, t)
}

// TestPermissionsAdminUserMgmtViewerDeniedWithoutPermission verifies that an admin user
// with no user mgmt permission receives 403 when calling even GET user mgmt endpoints.
func TestPermissionsAdminUserMgmtViewerDeniedWithoutPermission(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)
	// Give only flowviewer — no user mgmt permission.
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDFlowViewer})

	listUsersResp, err := adminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusForbidden, t)
}

// TestPermissionsAdminUserMgmtViewerGetSelf verifies that an admin user
// with no user mgmt permission can still get their own user information.
func TestPermissionsAdminUserMgmtViewerGetSelf(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)

	name := username()
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "pass"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	userRequestEditor := adminUserLogin(t, name, "pass")

	listUsersResp, err := adminClient.GetUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(userRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)
	matches(listUsersResp.JSON200.Username, name, t)
	matches(listUsersResp.JSON200.Id, createResp.JSON201.Id, t)
}

// TestPermissionsNormalUserLogoutSuper verifies that a normal admin user, even with permissions to call the logout
// endpoint, cannot log out the superuser.
func TestPermissionsNormalUserLogoutSuper(t *testing.T) {
	superRequestEditor := superLogin(t)
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDAdminUserMgmtViewer})

	// Normal admin users should not be able to log out the superuser, even if they
	// have permissions to call the logout endpoint.
	logoutResp, err := adminClient.LogoutSuperuserWithResponse(
		t.Context(), adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(logoutResp.StatusCode(), http.StatusForbidden, t)
}

// TestPermissionsAdminUserChangePasswordWrongUser verifies that an admin user cannot change another user's
// password without the appropriate permission.
func TestPermissionsAdminUserChangePasswordWrongUser(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)

	name := username()
	const pass = "correctpassword123"

	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	name2 := username()
	createResp2, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name2, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createResp2.StatusCode(), http.StatusCreated, t)

	userRequestEditor := adminUserLogin(t, name2, pass)

	changeResp, err := adminClient.ChangeUserPasswordWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.ChangeUserPasswordJSONRequestBody{OldPassword: pass, NewPassword: "newpass"},
		adminapi.RequestEditorFn(userRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(changeResp.StatusCode(), http.StatusForbidden, t)
	verifyAdminAPIErrorResponse(changeResp.JSON403, t)
}

// --- debugger permission ---

// TestPermissionsDebuggerAllowed verifies that an admin user with the debugger permission
// can call StartDebugSession.
func TestPermissionsDebuggerAllowed(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDDebugger})

	resp, err := adminClient.StartDebugSessionWithResponse(
		t.Context(),
		"echo",
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusOK, t)

	// Clean up.
	deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		resp.JSON200.Id,
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestPermissionsDebuggerDenied verifies that an admin user without the debugger permission
// receives 403 when calling StartDebugSession.
func TestPermissionsDebuggerDenied(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)
	// Give only flowviewer — no debugger.
	adminRequestEditor := createAdminUserInGroup(t, superRequestEditor, []int{PermissionIDFlowViewer})

	resp, err := adminClient.StartDebugSessionWithResponse(
		t.Context(),
		"echo",
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusForbidden, t)
	verifyAdminAPIErrorResponse(resp.JSON403, t)
}
