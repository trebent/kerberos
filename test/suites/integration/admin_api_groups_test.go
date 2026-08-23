package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

// TestAdminGroupCreate verifies that a new admin group can be created.
func TestAdminGroupCreate(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.GroupName()
	createResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	lib.Matches(createResp.JSON201.Name, name, t)
	if createResp.JSON201.Id == 0 {
		t.Fatal("expected non-zero group ID in create response")
	}

	if createResp.JSON201.Permissions == nil || len(*createResp.JSON201.Permissions) != len(lib.AllPermissionIDs) {
		got := 0
		if createResp.JSON201.Permissions != nil {
			got = len(*createResp.JSON201.Permissions)
		}
		t.Fatalf("expected %d permissions in create response, got %d",
			len(lib.AllPermissionIDs), got)
	}
}

// TestAdminGroupCreateConflict verifies that creating a duplicate admin group name is rejected.
func TestAdminGroupCreateConflict(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.GroupName()
	createResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	dupResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(dupResp.StatusCode(), http.StatusConflict, t)
	lib.VerifyAdminAPIErrorResponse(dupResp.JSON409, t)
}

// TestAdminGroupList verifies that a newly created admin group appears in the list response.
func TestAdminGroupList(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.GroupName()
	createResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	createdID := createResp.JSON201.Id

	listResp, err := lib.AdminClient.GetGroupsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusOK, t)
	for _, g := range *listResp.JSON200 {
		if g.Id == createdID {
			lib.Matches(g.Name, name, t)
			return
		}
	}
	t.Fatalf("admin group %d (%q) not found in list", createdID, name)
}

// TestAdminGroupGet verifies that a created admin group can be fetched by ID.
func TestAdminGroupGet(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.GroupName()
	createResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	groupID := createResp.JSON201.Id

	getResp, err := lib.AdminClient.GetGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Id, groupID, t)
	lib.Matches(getResp.JSON200.Name, name, t)
}

// TestAdminGroupGetNotFound verifies that fetching a non-existent admin group returns 404.
func TestAdminGroupGetNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	getResp, err := lib.AdminClient.GetGroupWithResponse(
		t.Context(),
		999999999,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(getResp.JSON404, t)
}

// TestAdminGroupUpdate verifies that an admin group's name can be updated.
func TestAdminGroupUpdate(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.GroupName()
	createResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	groupID := createResp.JSON201.Id

	newName := lib.GroupName()
	updateResp, err := lib.AdminClient.UpdateGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.UpdateGroupJSONRequestBody{Name: newName, PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.AdminClient.GetGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Name, newName, t)
}

// TestAdminGroupUpdateConflict verifies that updating an admin group's name to an existing name returns a conflict.
func TestAdminGroupUpdateConflict(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.GroupName()
	createResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	name2 := lib.GroupName()
	createResp2, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name2, PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp2.StatusCode(), http.StatusCreated, t)

	groupID := createResp.JSON201.Id

	updateResp, err := lib.AdminClient.UpdateGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.UpdateGroupJSONRequestBody{Name: name2, PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusConflict, t)
	lib.VerifyAdminAPIErrorResponse(updateResp.JSON409, t)
}

// TestAdminGroupDelete verifies that an admin group can be deleted and is no longer retrievable.
func TestAdminGroupDelete(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.GroupName()
	createResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: lib.AllPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	groupID := createResp.JSON201.Id

	deleteResp, err := lib.AdminClient.DeleteGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.AdminClient.GetGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestAdminGroupDeleteNotFound verifies that deleting a non-existent admin group returns 404.
func TestAdminGroupDeleteNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	deleteResp, err := lib.AdminClient.DeleteGroupWithResponse(
		t.Context(),
		999999999,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(deleteResp.JSON404, t)
}
