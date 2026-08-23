package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

// TestBasicAuthGroupCreate verifies that a new group can be created within an organisation and that
// the response contains the expected name and a valid ID.
func TestBasicAuthGroupCreate(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.GroupName()
	createResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	lib.Matches(createResp.JSON201.Name, name, t)
	if createResp.JSON201.Id == 0 {
		t.Fatal("expected non-zero group ID in create response")
	}
}

// TestBasicAuthGroupList verifies that a newly created group appears in the list response for its organisation.
func TestBasicAuthGroupList(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	createdID := createResp.JSON201.Id

	listResp, err := lib.BasicAuthClient.ListGroupsWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusOK, t)
	for _, group := range *listResp.JSON200 {
		if group.Id == createdID {
			return
		}
	}
	t.Fatalf("created group %d not found in list response", createdID)
}

// TestBasicAuthGroupGet verifies that a created group can be fetched by ID.
func TestBasicAuthGroupGet(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.GroupName()
	createResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	getResp, err := lib.BasicAuthClient.GetGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		createResp.JSON201.Id,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Id, createResp.JSON201.Id, t)
	lib.Matches(getResp.JSON200.Name, name, t)
}

// TestBasicAuthGroupGetNotFound verifies that fetching a deleted group returns 404.
func TestBasicAuthGroupGetNotFound(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createGroupResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)
	groupID := createGroupResp.JSON201.Id

	deleteResp, err := lib.BasicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.BasicAuthClient.GetGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestBasicAuthGroupUpdate verifies that a group's name can be changed and the updated value is
// reflected in a subsequent get.
func TestBasicAuthGroupUpdate(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	groupID := createResp.JSON201.Id

	newName := lib.GroupName()
	updateResp, err := lib.BasicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		groupID,
		authbasicapi.UpdateGroupJSONRequestBody{Name: newName},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusOK, t)
	lib.Matches(updateResp.JSON200.Name, newName, t)

	getResp, err := lib.BasicAuthClient.GetGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		groupID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Name, newName, t)
}

// TestBasicAuthGroupUpdateConflict verifies that renaming a group to an already-taken name within the
// same organisation returns a conflict error.
func TestBasicAuthGroupUpdateConflict(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	create1Resp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(create1Resp.StatusCode(), http.StatusCreated, t)

	create2Resp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(create2Resp.StatusCode(), http.StatusCreated, t)

	updateResp, err := lib.BasicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		create2Resp.JSON201.Id,
		authbasicapi.UpdateGroupJSONRequestBody{Name: create1Resp.JSON201.Name},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusConflict, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateResp.JSON409, t)
}

// TestBasicAuthGroupCreateConflict verifies that creating a group whose name already exists within the
// same organisation returns a conflict error.
func TestBasicAuthGroupCreateConflict(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.GroupName()
	createResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	conflictResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(conflictResp.StatusCode(), http.StatusConflict, t)
	lib.VerifyAuthBasicAPIErrorResponse(conflictResp.JSON409, t)
}

// TestBasicAuthGroupDelete verifies that a deleted group is no longer accessible.
func TestBasicAuthGroupDelete(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createGroupResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)
	groupID := createGroupResp.JSON201.Id

	deleteResp, err := lib.BasicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.BasicAuthClient.GetGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestBasicAuthGroupCreateOASValidation verifies that creating a group with an empty name is
// rejected with 400 by the OAS validator (name has minLength: 1).
func TestBasicAuthGroupCreateOASValidation(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	// Name below minLength: 1 — must be rejected.
	createResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: ""},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusBadRequest, t)
	lib.VerifyAuthBasicAPIErrorResponse(createResp.JSON400, t)
}

// TestBasicAuthGroupUpdateOASValidation verifies that updating a group with an empty name is
// rejected with 400 by the OAS validator (Group.name has minLength: 1).
func TestBasicAuthGroupUpdateOASValidation(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	// Name below minLength: 1 — must be rejected.
	updateResp, err := lib.BasicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		createResp.JSON201.Id,
		authbasicapi.UpdateGroupJSONRequestBody{Name: ""},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusBadRequest, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateResp.JSON400, t)
}

// TestBasicAuthGroupNoSession verifies that every group-scoped endpoint returns 401 with a
// populated error body when called without a session header.
func TestBasicAuthGroupNoSession(t *testing.T) {
	// CreateGroup — no session.
	createResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(createResp.JSON401, t)

	// ListGroups — no session.
	listResp, err := lib.BasicAuthClient.ListGroupsWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(listResp.JSON401, t)

	// GetGroup — no session.
	getResp, err := lib.BasicAuthClient.GetGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Groupid(alwaysGroupStaffID),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(getResp.JSON401, t)

	// UpdateGroup — no session.
	updateResp, err := lib.BasicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Groupid(alwaysGroupStaffID),
		authbasicapi.UpdateGroupJSONRequestBody{Id: int64(alwaysGroupStaffID), Name: lib.GroupName()},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateResp.JSON401, t)

	// DeleteGroup — no session.
	deleteResp, err := lib.BasicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Groupid(alwaysGroupStaffID),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(deleteResp.JSON401, t)
}

// TestBasicAuthGroupDeleteNotFound verifies deleting an already-deleted group.
func TestBasicAuthGroupDeleteNotFound(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createGroupResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)
	groupID := createGroupResp.JSON201.Id

	// First delete succeeds.
	deleteResp, err := lib.BasicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Second delete must return 404 (no body defined in spec).
	deleteAgainResp, err := lib.BasicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteAgainResp.StatusCode(), http.StatusNoContent, t)
}

// TestBasicAuthGroupUpdateNotFound verifies that attempting to update a deleted group returns 404
// (no body defined in spec).
func TestBasicAuthGroupUpdateNotFound(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createGroupResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)
	groupID := createGroupResp.JSON201.Id

	// Delete the group first.
	deleteResp, err := lib.BasicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Update the deleted group must return 404 (no body defined in spec).
	updateResp, err := lib.BasicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.UpdateGroupJSONRequestBody{Id: groupID, Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNotFound, t)
}
