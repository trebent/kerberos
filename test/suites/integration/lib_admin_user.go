package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
)

// mustGetAdminUserID fetches the admin user list and returns the ID of the user with the given username.
func mustGetAdminUserID(t *testing.T, requestEditor RequestEditorFn, name string) int {
	t.Helper()
	resp, err := adminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusOK, t)
	for _, u := range *resp.JSON200 {
		if u.Username == name {
			return u.Id
		}
	}
	t.Fatalf("admin user %q not found in list", name)
	return 0
}

// createAdminUserInGroup creates a fresh admin user, creates a group with the specified
// permissionIDs, adds the user to that group, and returns the user's session.
func createAdminUserInGroup(t *testing.T, requestEditor RequestEditorFn, permissionIDs []int) RequestEditorFn {
	t.Helper()

	const pass = "testpassword1"
	name := username()

	createUserResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(requestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)

	userID := mustGetAdminUserID(t, requestEditor, name)

	grpResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: permissionIDs},
		adminapi.RequestEditorFn(requestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(grpResp.StatusCode(), http.StatusCreated, t)

	updateResp, err := adminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grpResp.JSON201.Id}},
		adminapi.RequestEditorFn(requestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	return adminUserLogin(t, name, pass)
}
