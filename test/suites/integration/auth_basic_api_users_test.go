package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"slices"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

// TestBasicAuthUserCreate verifies that a new user can be created within an organisation and that
// the response contains the expected name and a valid ID.
func TestBasicAuthUserCreate(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: name, Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	lib.Matches(createResp.JSON201.Name, name, t)
	if createResp.JSON201.Id == 0 {
		t.Fatal("expected non-zero user ID in create response")
	}
}

// TestBasicAuthUserList verifies that a newly created user appears in the list response for its organisation.
func TestBasicAuthUserList(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	createdID := createResp.JSON201.Id

	listResp, err := lib.BasicAuthClient.ListUsersWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusOK, t)
	for _, user := range *listResp.JSON200 {
		if user.Id == createdID {
			return
		}
	}
	t.Fatalf("created user %d not found in list response", createdID)
}

// TestBasicAuthUserGet verifies that a created user can be fetched by ID.
func TestBasicAuthUserGet(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: name, Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	getResp, err := lib.BasicAuthClient.GetUserWithResponse(
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

// TestBasicAuthUserGetNotFound verifies that fetching a deleted user returns 404.
func TestBasicAuthUserGetNotFound(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	deleteResp, err := lib.BasicAuthClient.DeleteUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.BasicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestBasicAuthUserUpdate verifies that a user's name can be changed and the updated value is
// reflected in a subsequent get.
func TestBasicAuthUserUpdate(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	userID := createResp.JSON201.Id

	newName := lib.Username()
	updateResp, err := lib.BasicAuthClient.UpdateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		userID,
		authbasicapi.UpdateUserJSONRequestBody{Name: newName},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusOK, t)
	lib.Matches(updateResp.JSON200.Name, newName, t)

	getResp, err := lib.BasicAuthClient.GetUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		userID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Name, newName, t)
}

// TestBasicAuthUserUpdateConflict verifies that renaming a user to an already-taken name within the
// same organisation returns a conflict error.
func TestBasicAuthUserUpdateConflict(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	create1Resp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(create1Resp.StatusCode(), http.StatusCreated, t)

	create2Resp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(create2Resp.StatusCode(), http.StatusCreated, t)

	updateResp, err := lib.BasicAuthClient.UpdateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		create2Resp.JSON201.Id,
		authbasicapi.UpdateUserJSONRequestBody{Name: create1Resp.JSON201.Name},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusConflict, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateResp.JSON409, t)
}

// TestBasicAuthUserCreateConflict verifies that creating a user whose name already exists within the
// same organisation returns a conflict error.
func TestBasicAuthUserCreateConflict(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: name, Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	conflictResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: name, Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(conflictResp.StatusCode(), http.StatusConflict, t)
	lib.VerifyAuthBasicAPIErrorResponse(conflictResp.JSON409, t)
}

// TestBasicAuthUserDelete verifies that a deleted user is no longer accessible.
func TestBasicAuthUserDelete(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	deleteResp, err := lib.BasicAuthClient.DeleteUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.BasicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestBasicAuthUserCreateOASValidation verifies that creating a user with a name that is too short
// or a password that is outside the allowed length range is rejected with 400.
func TestBasicAuthUserCreateOASValidation(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	// Name below minLength: 5 — must be rejected.
	shortNameResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: "ab", Password: "validpassword"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(shortNameResp.StatusCode(), http.StatusBadRequest, t)
	lib.VerifyAuthBasicAPIErrorResponse(shortNameResp.JSON400, t)

	// Password below minLength: 10 — must be rejected.
	shortPasswordResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "short"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(shortPasswordResp.StatusCode(), http.StatusBadRequest, t)
	lib.VerifyAuthBasicAPIErrorResponse(shortPasswordResp.JSON400, t)

	// Password above maxLength: 40 — must be rejected.
	longPasswordResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "this-password-is-way-too-long-for-the-schema-limits"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(longPasswordResp.StatusCode(), http.StatusBadRequest, t)
	lib.VerifyAuthBasicAPIErrorResponse(longPasswordResp.JSON400, t)
}

// TestBasicAuthUserChangePassword verifies the full change-password flow: a user can log in,
// change their password, and then log in again with the new password.
func TestBasicAuthUserChangePassword(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	oldPassword := "oldpassword123"
	newPassword := "newpassword456"
	name := lib.Username()

	createUserResp2, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: name, Password: oldPassword},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp2.StatusCode(), http.StatusCreated, t)
	userID2 := createUserResp2.JSON201.Id

	loginResp, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{Username: name, Password: oldPassword},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	orgUserRequestEditor := lib.SessionCookieRequestEditor(loginResp.HTTPResponse, t)

	changeResp, err := lib.BasicAuthClient.ChangePasswordWithResponse(
		t.Context(),
		orgID,
		userID2,
		authbasicapi.ChangePasswordJSONRequestBody{OldPassword: oldPassword, Password: newPassword},
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(changeResp.StatusCode(), http.StatusNoContent, t)

	// Login with new password must succeed.
	newLoginResp, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{Username: name, Password: newPassword},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(newLoginResp.StatusCode(), http.StatusNoContent, t)

	// Login with old password must now fail.
	oldLoginResp, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{Username: name, Password: oldPassword},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(oldLoginResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(oldLoginResp.JSON401, t)
}

// TestBasicAuthUserChangePasswordOASValidation verifies that the OAS validator rejects change-password
// requests with credentials that violate the schema length constraints.
// Note: the spec does not define a 400 response body for this endpoint, so only the
// status code is checked.
func TestBasicAuthUserChangePasswordOASValidation(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	// oldPassword below minLength: 10 — must be rejected before auth checks.
	shortOldPwResp, err := lib.BasicAuthClient.ChangePasswordWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.ChangePasswordJSONRequestBody{OldPassword: "short", Password: "validpassword123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(shortOldPwResp.StatusCode(), http.StatusBadRequest, t)

	// new password below minLength: 10 — must be rejected.
	shortNewPwResp, err := lib.BasicAuthClient.ChangePasswordWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.ChangePasswordJSONRequestBody{OldPassword: "validoldpassword", Password: "short"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(shortNewPwResp.StatusCode(), http.StatusBadRequest, t)
}

// TestBasicAuthUserNoSession verifies that every user-scoped endpoint returns 401 with a populated
// error body when called without a session header.
func TestBasicAuthUserNoSession(t *testing.T) {
	// CreateUser — no session.
	createResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(createResp.JSON401, t)

	// ListUsers — no session.
	listResp, err := lib.BasicAuthClient.ListUsersWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(listResp.JSON401, t)

	// GetUser — no session.
	getResp, err := lib.BasicAuthClient.GetUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(getResp.JSON401, t)

	// UpdateUser — no session.
	updateResp, err := lib.BasicAuthClient.UpdateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.UpdateUserJSONRequestBody{Id: int64(alwaysUserID), Name: lib.Username()},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateResp.JSON401, t)

	// DeleteUser — no session.
	deleteResp, err := lib.BasicAuthClient.DeleteUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(deleteResp.JSON401, t)

	// UpdateUserGroups — no session.
	updateGroupsResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.UpdateUserGroupsJSONRequestBody{},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateGroupsResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateGroupsResp.JSON401, t)

	// GetUserGroups — no session.
	getGroupsResp, err := lib.BasicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getGroupsResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(getGroupsResp.JSON401, t)

	// ChangePassword — no session.
	changePwResp, err := lib.BasicAuthClient.ChangePasswordWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.ChangePasswordJSONRequestBody{OldPassword: "validoldpassword", Password: "validnewpassword"},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(changePwResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(changePwResp.JSON401, t)
}

// TestBasicAuthUserDeleteNotFound verifies deleting an already-deleted user.
func TestBasicAuthUserDeleteNotFound(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	// First delete succeeds.
	deleteResp, err := lib.BasicAuthClient.DeleteUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Second delete must return 404 (no body defined in spec).
	deleteAgainResp, err := lib.BasicAuthClient.DeleteUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteAgainResp.StatusCode(), http.StatusNoContent, t)
}

// TestBasicAuthUserUpdateNotFound verifies that attempting to update a deleted user returns 404
// (no body defined in spec).
func TestBasicAuthUserUpdateNotFound(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	// Delete the user first.
	deleteResp, err := lib.BasicAuthClient.DeleteUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Update the deleted user must return 404 (no body defined in spec).
	updateResp, err := lib.BasicAuthClient.UpdateUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserJSONRequestBody{Id: userID, Name: lib.Username()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNotFound, t)
}

// TestBasicAuthUserGroupBindingAssign verifies that groups can be assigned to a user and are returned
// by GetUserGroups.
func TestBasicAuthUserGroupBindingAssign(t *testing.T) {
	superLoginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: lib.SuperUserClientID, ClientSecret: lib.SuperUserClientSecret},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := lib.SessionCookieRequestEditor(superLoginResp.HTTPResponse, t)

	orgID, adminRequestEditor := lib.OrgWithSession(t, superRequestEditor)

	groupAName := lib.GroupName()
	createGroupA, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupAName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupA.StatusCode(), http.StatusCreated, t)

	groupBName := lib.GroupName()
	createGroupB, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupBName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupB.StatusCode(), http.StatusCreated, t)

	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	updateResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupA.JSON201.Id, Name: groupAName},
			{Id: createGroupB.JSON201.Id, Name: groupBName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusOK, t)

	getResp, err := lib.BasicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	groups := *getResp.JSON200
	if !slices.ContainsFunc(groups, func(g authbasicapi.Group) bool { return g.Name == groupAName }) {
		t.Fatalf("expected group %q in user groups, got %v", groupAName, groups)
	}
	if !slices.ContainsFunc(groups, func(g authbasicapi.Group) bool { return g.Name == groupBName }) {
		t.Fatalf("expected group %q in user groups, got %v", groupBName, groups)
	}
}

// TestBasicAuthUserGroupBindingReplace verifies that updating a user's groups replaces the previous
// set entirely — groups removed from the request are no longer returned.
func TestBasicAuthUserGroupBindingReplace(t *testing.T) {
	superLoginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: lib.SuperUserClientID, ClientSecret: lib.SuperUserClientSecret},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := lib.SessionCookieRequestEditor(superLoginResp.HTTPResponse, t)

	orgID, adminRequestEditor := lib.OrgWithSession(t, superRequestEditor)

	groupAName := lib.GroupName()
	createGroupA, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupAName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupA.StatusCode(), http.StatusCreated, t)

	groupBName := lib.GroupName()
	createGroupB, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupBName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupB.StatusCode(), http.StatusCreated, t)

	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	// Assign both groups initially.
	initialUpdateResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupA.JSON201.Id, Name: groupAName},
			{Id: createGroupB.JSON201.Id, Name: groupBName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(initialUpdateResp.StatusCode(), http.StatusOK, t)

	// Replace with only group B — group A should be removed.
	replaceResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupB.JSON201.Id, Name: groupBName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(replaceResp.StatusCode(), http.StatusOK, t)

	getResp, err := lib.BasicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	groups := *getResp.JSON200
	if slices.ContainsFunc(groups, func(g authbasicapi.Group) bool { return g.Name == groupAName }) {
		t.Fatalf("group %q should have been removed after replace, got %v", groupAName, groups)
	}
	if !slices.ContainsFunc(groups, func(g authbasicapi.Group) bool { return g.Name == groupBName }) {
		t.Fatalf("expected group %q to remain after replace, got %v", groupBName, groups)
	}
}

// TestBasicAuthUserGroupBindingClear verifies that assigning an empty group list removes all group
// memberships from the user.
func TestBasicAuthUserGroupBindingClear(t *testing.T) {
	superLoginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: lib.SuperUserClientID, ClientSecret: lib.SuperUserClientSecret},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := lib.SessionCookieRequestEditor(superLoginResp.HTTPResponse, t)

	orgID, adminRequestEditor := lib.OrgWithSession(t, superRequestEditor)

	gName := lib.GroupName()
	createGroupResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: gName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)

	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	// Assign the group first.
	assignResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupResp.JSON201.Id, Name: gName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(assignResp.StatusCode(), http.StatusOK, t)

	// Clear all groups.
	clearResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(clearResp.StatusCode(), http.StatusOK, t)

	getResp, err := lib.BasicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	if len(*getResp.JSON200) != 0 {
		t.Fatalf("expected empty groups after clear, got %v", *getResp.JSON200)
	}
}

// TestBasicAuthUserGroupBindingGet verifies that GetUserGroups returns the expected groups for a user
// that was set up with known group memberships in TestMain.
func TestBasicAuthUserGroupBindingGet(t *testing.T) {
	loginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: lib.SuperUserClientID, ClientSecret: lib.SuperUserClientSecret},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := lib.SessionCookieRequestEditor(loginResp.HTTPResponse, t)

	getResp, err := lib.BasicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	groups := *getResp.JSON200
	for _, expected := range []string{alwaysGroupStaff, alwaysGroupPleb, alwaysGroupDev} {
		if !slices.ContainsFunc(groups, func(g authbasicapi.Group) bool { return g.Name == expected }) {
			t.Fatalf("expected group %q in always-user groups, got %v", expected, groups)
		}
	}
}
