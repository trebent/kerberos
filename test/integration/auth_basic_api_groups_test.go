package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/ft/client/admin"
	authbasicapi "github.com/trebent/kerberos/ft/client/auth/basic"
)

// TestGroupCreate verifies that a new group can be created within an organisation and that
// the response contains the expected name and a valid ID.
func TestGroupCreate(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	name := groupName()
	createResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	matches(createResp.JSON201.Name, name, t)
	if createResp.JSON201.Id == 0 {
		t.Fatal("expected non-zero group ID in create response")
	}
}

// TestGroupList verifies that a newly created group appears in the list response for its organisation.
func TestGroupList(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	createResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	createdID := createResp.JSON201.Id

	listResp, err := basicAuthClient.ListGroupsWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusOK, t)
	for _, group := range *listResp.JSON200 {
		if group.Id == createdID {
			return
		}
	}
	t.Fatalf("created group %d not found in list response", createdID)
}

// TestGroupGet verifies that a created group can be fetched by ID.
func TestGroupGet(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	name := groupName()
	createResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	getResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		createResp.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	matches(getResp.JSON200.Id, createResp.JSON201.Id, t)
	matches(getResp.JSON200.Name, name, t)
}

// TestGroupGetNotFound verifies that fetching a deleted group returns 404.
func TestGroupGetNotFound(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createGroupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)
	groupID := createGroupResp.JSON201.Id

	deleteResp, err := basicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestGroupUpdate verifies that a group's name can be changed and the updated value is
// reflected in a subsequent get.
func TestGroupUpdate(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	createResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	groupID := createResp.JSON201.Id

	newName := groupName()
	updateResp, err := basicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		groupID,
		authbasicapi.UpdateGroupJSONRequestBody{Name: newName},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusOK, t)
	matches(updateResp.JSON200.Name, newName, t)

	getResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	matches(getResp.JSON200.Name, newName, t)
}

// TestGroupUpdateConflict verifies that renaming a group to an already-taken name within the
// same organisation returns a conflict error.
func TestGroupUpdateConflict(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	create1Resp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(create1Resp.StatusCode(), http.StatusCreated, t)

	create2Resp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(create2Resp.StatusCode(), http.StatusCreated, t)

	updateResp, err := basicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		create2Resp.JSON201.Id,
		authbasicapi.UpdateGroupJSONRequestBody{Name: create1Resp.JSON201.Name},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusConflict, t)
	verifyAuthBasicAPIErrorResponse(updateResp.JSON409, t)
}

// TestGroupCreateConflict verifies that creating a group whose name already exists within the
// same organisation returns a conflict error.
func TestGroupCreateConflict(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	name := groupName()
	createResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	conflictResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(conflictResp.StatusCode(), http.StatusConflict, t)
	verifyAuthBasicAPIErrorResponse(conflictResp.JSON409, t)
}

// TestGroupDelete verifies that a deleted group is no longer accessible.
func TestGroupDelete(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createGroupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)
	groupID := createGroupResp.JSON201.Id

	deleteResp, err := basicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestGroupCreateOASValidation verifies that creating a group with an empty name is
// rejected with 400 by the OAS validator (name has minLength: 1).
func TestGroupCreateOASValidation(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	// Name below minLength: 1 — must be rejected.
	createResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: ""},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusBadRequest, t)
	verifyAuthBasicAPIErrorResponse(createResp.JSON400, t)
}

// TestGroupUpdateOASValidation verifies that updating a group with an empty name is
// rejected with 400 by the OAS validator (Group.name has minLength: 1).
func TestGroupUpdateOASValidation(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	createResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	// Name below minLength: 1 — must be rejected.
	updateResp, err := basicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		createResp.JSON201.Id,
		authbasicapi.UpdateGroupJSONRequestBody{Name: ""},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusBadRequest, t)
	verifyAuthBasicAPIErrorResponse(updateResp.JSON400, t)
}

// TestGroupNoSession verifies that every group-scoped endpoint returns 401 with a
// populated error body when called without a session header.
func TestGroupNoSession(t *testing.T) {
	// CreateGroup — no session.
	createResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(createResp.JSON401, t)

	// ListGroups — no session.
	listResp, err := basicAuthClient.ListGroupsWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(listResp.JSON401, t)

	// GetGroup — no session.
	getResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Groupid(alwaysGroupStaffID),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(getResp.JSON401, t)

	// UpdateGroup — no session.
	updateResp, err := basicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Groupid(alwaysGroupStaffID),
		authbasicapi.UpdateGroupJSONRequestBody{Id: int64(alwaysGroupStaffID), Name: groupName()},
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(updateResp.JSON401, t)

	// DeleteGroup — no session.
	deleteResp, err := basicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Groupid(alwaysGroupStaffID),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(deleteResp.JSON401, t)
}

// TestGroupDeleteNotFound verifies deleting an already-deleted group.
func TestGroupDeleteNotFound(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createGroupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)
	groupID := createGroupResp.JSON201.Id

	// First delete succeeds.
	deleteResp, err := basicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Second delete must return 404 (no body defined in spec).
	deleteAgainResp, err := basicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteAgainResp.StatusCode(), http.StatusNoContent, t)
}

// TestGroupUpdateNotFound verifies that attempting to update a deleted group returns 404
// (no body defined in spec).
func TestGroupUpdateNotFound(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createGroupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)
	groupID := createGroupResp.JSON201.Id

	// Delete the group first.
	deleteResp, err := basicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Update the deleted group must return 404 (no body defined in spec).
	updateResp, err := basicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.UpdateGroupJSONRequestBody{Id: groupID, Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNotFound, t)
}
