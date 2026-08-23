package lib

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

// MustGetAdminUserID fetches the admin user list and returns the ID of the user with the given username.
func MustGetAdminUserID(t *testing.T, requestEditor RequestEditorFn, name string) int {
	t.Helper()
	resp, err := AdminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditor),
	)
	CheckErr(err, t)
	VerifyStatusCode(resp.StatusCode(), http.StatusOK, t)
	for _, u := range *resp.JSON200 {
		if u.Username == name {
			return u.Id
		}
	}
	t.Fatalf("admin user %q not found in list", name)
	return 0
}

// CreateAdminUserInGroup creates a fresh admin user, creates a group with the specified
// permissionIDs, adds the user to that group, and returns the user's session.
func CreateAdminUserInGroup(t *testing.T, requestEditor RequestEditorFn, permissionIDs []int) RequestEditorFn {
	t.Helper()

	const pass = "testpassword1"
	name := Username()

	createUserResp, err := AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(requestEditor),
	)
	CheckErr(err, t)
	VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)

	userID := MustGetAdminUserID(t, requestEditor, name)

	grpResp, err := AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: GroupName(), PermissionIDs: permissionIDs},
		adminapi.RequestEditorFn(requestEditor),
	)
	CheckErr(err, t)
	VerifyStatusCode(grpResp.StatusCode(), http.StatusCreated, t)

	updateResp, err := AdminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grpResp.JSON201.Id}},
		adminapi.RequestEditorFn(requestEditor),
	)
	CheckErr(err, t)
	VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	return AdminUserLogin(t, name, pass)
}
