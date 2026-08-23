package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

func TestAdminGetBackendOASNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	resp, err := lib.AdminClient.GetBackendOASWithResponse(
		t.Context(),
		"nonexistent-backend",
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(resp.JSON404, t)
}

func TestAdminGetBackendOAS(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	getBackendOASResp, err := lib.AdminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getBackendOASResp.StatusCode(), http.StatusOK, t)
}

// TestAdminGetBackendOASAsAdminUser verifies that a non-superuser admin user can also access the GetBackendOAS endpoint.
func TestAdminGetBackendOASAsAdminUser(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	const pass = "password123"
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	userID := lib.MustGetAdminUserID(t, superRequestEditor, name)

	// Create a group with the oasviewer permission and assign the user to it.
	grpResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(grpResp.StatusCode(), http.StatusCreated, t)

	updateResp, err := lib.AdminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grpResp.JSON201.Id}},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	adminRequestEditor := lib.AdminUserLogin(t, name, pass)

	getBackendOASResp, err := lib.AdminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getBackendOASResp.StatusCode(), http.StatusOK, t)
}
