package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
)

// TestAdminGroupCreate verifies that a new admin group can be created.
func TestAdminGroupCreate(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := groupName()
	createResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	matches(createResp.JSON201.Name, name, t)
	if createResp.JSON201.Id == 0 {
		t.Fatal("expected non-zero group ID in create response")
	}

	if len(createResp.JSON201.Permissions) != len(allPermissionIDs) {
		t.Fatalf("expected %d permissions in create response, got %d",
			len(allPermissionIDs), len(createResp.JSON201.Permissions))
	}
}

// TestAdminGroupCreateConflict verifies that creating a duplicate admin group name is rejected.
func TestAdminGroupCreateConflict(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := groupName()
	createResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	dupResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(dupResp.StatusCode(), http.StatusConflict, t)
	verifyAdminAPIErrorResponse(dupResp.JSON409, t)
}

// TestAdminGroupList verifies that a newly created admin group appears in the list response.
func TestAdminGroupList(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := groupName()
	createResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	createdID := createResp.JSON201.Id

	listResp, err := adminClient.GetGroupsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusOK, t)
	for _, g := range *listResp.JSON200 {
		if g.Id == createdID {
			matches(g.Name, name, t)
			return
		}
	}
	t.Fatalf("admin group %d (%q) not found in list", createdID, name)
}

// TestAdminGroupGet verifies that a created admin group can be fetched by ID.
func TestAdminGroupGet(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := groupName()
	createResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	groupID := createResp.JSON201.Id

	getResp, err := adminClient.GetGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	matches(getResp.JSON200.Id, groupID, t)
	matches(getResp.JSON200.Name, name, t)
}

// TestAdminGroupGetNotFound verifies that fetching a non-existent admin group returns 404.
func TestAdminGroupGetNotFound(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	getResp, err := adminClient.GetGroupWithResponse(
		t.Context(),
		999999999,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
	verifyAdminAPIErrorResponse(getResp.JSON404, t)
}

// TestAdminGroupUpdate verifies that an admin group's name can be updated.
func TestAdminGroupUpdate(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := groupName()
	createResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	groupID := createResp.JSON201.Id

	newName := groupName()
	updateResp, err := adminClient.UpdateGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.UpdateGroupJSONRequestBody{Name: newName, PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := adminClient.GetGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	matches(getResp.JSON200.Name, newName, t)
}

// TestAdminGroupUpdateConflict verifies that updating an admin group's name to an existing name returns a conflict.
func TestAdminGroupUpdateConflict(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := groupName()
	createResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	name2 := groupName()
	createResp2, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name2, PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp2.StatusCode(), http.StatusCreated, t)

	groupID := createResp.JSON201.Id

	updateResp, err := adminClient.UpdateGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.UpdateGroupJSONRequestBody{Name: name2, PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusConflict, t)
	verifyAdminAPIErrorResponse(updateResp.JSON409, t)
}

// TestAdminGroupDelete verifies that an admin group can be deleted and is no longer retrievable.
func TestAdminGroupDelete(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := groupName()
	createResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: name, PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	groupID := createResp.JSON201.Id

	deleteResp, err := adminClient.DeleteGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := adminClient.GetGroupWithResponse(
		t.Context(),
		groupID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestAdminGroupDeleteNotFound verifies that deleting a non-existent admin group returns 404.
func TestAdminGroupDeleteNotFound(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	deleteResp, err := adminClient.DeleteGroupWithResponse(
		t.Context(),
		999999999,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNotFound, t)
	verifyAdminAPIErrorResponse(deleteResp.JSON404, t)
}
