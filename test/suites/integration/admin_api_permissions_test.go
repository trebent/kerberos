package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/integration/client/auth/basic"
)

// permIDs are the fixed permission IDs bootstrapped by the server.
const (
	permIDFlowViewer         = 1
	permIDOASViewer          = 2
	permIDBasicAuthOrgAdmin  = 3
	permIDBasicAuthOrgViewer = 4
)

// createAdminUserInGroup creates a fresh admin user, creates a group with the specified
// permissionIDs, adds the user to that group, and returns the user's session.
func createAdminUserInGroup(t *testing.T, superSession string, permissionIDs []int) string {
	t.Helper()

	const pass = "testpassword1"
	name := username()

	createUserResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)

	userID := mustGetAdminUserID(t, superSession, name)

	grpResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: permissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(grpResp.StatusCode(), http.StatusCreated, t)

	updateResp, err := adminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grpResp.JSON201.Id}},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	return adminUserLogin(t, name, pass)
}

// --- GetPermissions ---

// TestPermissionsGetPermissions verifies that any authenticated admin user can list
// available permissions.
func TestPermissionsGetPermissions(t *testing.T) {
	superSession := superLogin(t)

	resp, err := adminClient.GetPermissionsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
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
		permIDFlowViewer:         "flowviewer",
		permIDOASViewer:          "oasviewer",
		permIDBasicAuthOrgAdmin:  "basicauthorgadmin",
		permIDBasicAuthOrgViewer: "basicauthorgviewer",
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
	superSession := superLogin(t)

	// GetFlow — requires flowviewer.
	getFlowResp, err := adminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getFlowResp.StatusCode(), http.StatusOK, t)

	// GetBackendOAS — requires oasviewer.
	getOASResp, err := adminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getOASResp.StatusCode(), http.StatusOK, t)

	// Basic auth endpoint (GET) — requires basicauthorgadmin or basicauthorgviewer.
	orgID, _ := orgWithSession(t, superSession)
	listUsersResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// Basic auth endpoint (non-GET) — requires basicauthorgadmin.
	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
}

// --- flowviewer permission ---

// TestPermissionsFlowViewerAllowed verifies that an admin user with the flowviewer
// permission can call GetFlow.
func TestPermissionsFlowViewerAllowed(t *testing.T) {
	superSession := superLogin(t)
	session := createAdminUserInGroup(t, superSession, []int{permIDFlowViewer})

	resp, err := adminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusOK, t)
}

// TestPermissionsFlowViewerDeniedWithoutPermission verifies that an admin user without
// the flowviewer permission receives 403 when calling GetFlow.
func TestPermissionsFlowViewerDeniedWithoutPermission(t *testing.T) {
	superSession := superLogin(t)
	// Give only oasviewer — no flowviewer.
	session := createAdminUserInGroup(t, superSession, []int{permIDOASViewer})

	resp, err := adminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusForbidden, t)
}

// TestPermissionsFlowViewerDeniedNoGroup verifies that an admin user in no group at all
// receives 403 when calling GetFlow.
func TestPermissionsFlowViewerDeniedNoGroup(t *testing.T) {
	superSession := superLogin(t)

	const pass = "testpassword1"
	name := username()
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	session := adminUserLogin(t, name, pass)

	resp, err := adminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusForbidden, t)
}

// --- oasviewer permission ---

// TestPermissionsOASViewerAllowed verifies that an admin user with the oasviewer
// permission can call GetBackendOAS.
func TestPermissionsOASViewerAllowed(t *testing.T) {
	superSession := superLogin(t)
	session := createAdminUserInGroup(t, superSession, []int{permIDOASViewer})

	resp, err := adminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusOK, t)
}

// TestPermissionsOASViewerDeniedWithoutPermission verifies that an admin user without
// the oasviewer permission receives 403 when calling GetBackendOAS.
func TestPermissionsOASViewerDeniedWithoutPermission(t *testing.T) {
	superSession := superLogin(t)
	// Give only flowviewer — no oasviewer.
	session := createAdminUserInGroup(t, superSession, []int{permIDFlowViewer})

	resp, err := adminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusForbidden, t)
}

// --- basicauthorgadmin permission ---

// TestPermissionsBasicAuthOrgAdminAllowed verifies that an admin user with the
// basicauthorgadmin permission can perform both read and write operations on the
// basic auth API.
func TestPermissionsBasicAuthOrgAdminAllowed(t *testing.T) {
	superSession := superLogin(t)
	session := createAdminUserInGroup(t, superSession, []int{permIDBasicAuthOrgAdmin})

	// basicauthorgadmin must be able to create an organisation (write).
	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	orgID := createOrgResp.JSON201.Id

	// basicauthorgadmin must be able to list users (read).
	listUsersResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// basicauthorgadmin must be able to create a user (write).
	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
}

// TestPermissionsBasicAuthOrgAdminDeniedWithoutPermission verifies that an admin user
// without any basic auth permission cannot access the basic auth API. The middleware falls
// through to session lookup (which does not recognise an admin session), returning 401.
func TestPermissionsBasicAuthOrgAdminDeniedWithoutPermission(t *testing.T) {
	superSession := superLogin(t)
	// Give only flowviewer — no basic auth permission.
	session := createAdminUserInGroup(t, superSession, []int{permIDFlowViewer})

	// The admin session is not a valid basic auth session, so the middleware returns 401.
	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusUnauthorized, t)
}

// --- basicauthorgviewer permission ---

// TestPermissionsBasicAuthOrgViewerReadAllowed verifies that an admin user with the
// basicauthorgviewer permission can call GET endpoints on the basic auth API.
func TestPermissionsBasicAuthOrgViewerReadAllowed(t *testing.T) {
	superSession := superLogin(t)

	// Create an org via the superuser first so there is something to read.
	orgID, _ := orgWithSession(t, superSession)

	session := createAdminUserInGroup(t, superSession, []int{permIDBasicAuthOrgViewer})

	// basicauthorgviewer must be able to list users (GET).
	listUsersResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusOK, t)

	// basicauthorgviewer must be able to list groups (GET).
	listGroupsResp, err := basicAuthClient.ListGroupsWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(listGroupsResp.StatusCode(), http.StatusOK, t)
}

// TestPermissionsBasicAuthOrgViewerWriteDenied verifies that an admin user with the
// basicauthorgviewer permission is denied for non-GET (write) endpoints on the basic
// auth API.
func TestPermissionsBasicAuthOrgViewerWriteDenied(t *testing.T) {
	superSession := superLogin(t)
	session := createAdminUserInGroup(t, superSession, []int{permIDBasicAuthOrgViewer})

	// basicauthorgviewer must NOT be able to create an organisation (POST).
	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusForbidden, t)

	// Also verify that a user-creation call (POST) is denied.
	orgID, _ := orgWithSession(t, superSession)
	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusForbidden, t)
}

// TestPermissionsBasicAuthOrgViewerDeniedWithoutPermission verifies that an admin user
// with no basic auth permission cannot access even GET endpoints on the basic auth API.
// The middleware falls through to session lookup (which does not recognise an admin session),
// returning 401.
func TestPermissionsBasicAuthOrgViewerDeniedWithoutPermission(t *testing.T) {
	superSession := superLogin(t)
	orgID, _ := orgWithSession(t, superSession)

	// Give only flowviewer — no basic auth permission.
	session := createAdminUserInGroup(t, superSession, []int{permIDFlowViewer})

	listUsersResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusUnauthorized, t)
}

// --- Group response includes permissions ---

// TestPermissionsGroupResponseIncludesPermissions verifies that the permissions field is
// present and accurate in the group create/get responses.
func TestPermissionsGroupResponseIncludesPermissions(t *testing.T) {
	superSession := superLogin(t)

	permIDs := []int{permIDFlowViewer, permIDOASViewer}
	createResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: permIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
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
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
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
