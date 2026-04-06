package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/ft/client/admin"
	authbasicapi "github.com/trebent/kerberos/ft/client/auth/basic"
)

// TestUserCreate verifies that a new user can be created within an organisation and that
// the response contains the expected name and a valid ID.
func TestUserCreate(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	name := username()
	createResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: name, Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	matches(createResp.JSON201.Name, name, t)
	if createResp.JSON201.Id == 0 {
		t.Fatal("expected non-zero user ID in create response")
	}
}

// TestUserList verifies that a newly created user appears in the list response for its organisation.
func TestUserList(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	createResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	createdID := createResp.JSON201.Id

	listResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusOK, t)
	for _, user := range *listResp.JSON200 {
		if user.Id == createdID {
			return
		}
	}
	t.Fatalf("created user %d not found in list response", createdID)
}

// TestUserGet verifies that a created user can be fetched by ID.
func TestUserGet(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	name := username()
	createResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: name, Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	getResp, err := basicAuthClient.GetUserWithResponse(
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

// TestUserGetNotFound verifies that fetching a deleted user returns 404.
func TestUserGetNotFound(t *testing.T) {
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

	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	deleteResp, err := basicAuthClient.DeleteUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestUserUpdate verifies that a user's name can be changed and the updated value is
// reflected in a subsequent get.
func TestUserUpdate(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	createResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	userID := createResp.JSON201.Id

	newName := username()
	updateResp, err := basicAuthClient.UpdateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		userID,
		authbasicapi.UpdateUserJSONRequestBody{Name: newName},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusOK, t)
	matches(updateResp.JSON200.Name, newName, t)

	getResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	matches(getResp.JSON200.Name, newName, t)
}

// TestUserUpdateConflict verifies that renaming a user to an already-taken name within the
// same organisation returns a conflict error.
func TestUserUpdateConflict(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	create1Resp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(create1Resp.StatusCode(), http.StatusCreated, t)

	create2Resp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(create2Resp.StatusCode(), http.StatusCreated, t)

	updateResp, err := basicAuthClient.UpdateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		create2Resp.JSON201.Id,
		authbasicapi.UpdateUserJSONRequestBody{Name: create1Resp.JSON201.Name},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusConflict, t)
	verifyAuthBasicAPIErrorResponse(updateResp.JSON409, t)
}

// TestUserCreateConflict verifies that creating a user whose name already exists within the
// same organisation returns a conflict error.
func TestUserCreateConflict(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	name := username()
	createResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: name, Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	conflictResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: name, Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(conflictResp.StatusCode(), http.StatusConflict, t)
	verifyAuthBasicAPIErrorResponse(conflictResp.JSON409, t)
}

// TestUserDelete verifies that a deleted user is no longer accessible.
func TestUserDelete(t *testing.T) {
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

	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	deleteResp, err := basicAuthClient.DeleteUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestUserCreateOASValidation verifies that creating a user with a name that is too short
// or a password that is outside the allowed length range is rejected with 400.
func TestUserCreateOASValidation(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	// Name below minLength: 5 — must be rejected.
	shortNameResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: "ab", Password: "validpassword"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(shortNameResp.StatusCode(), http.StatusBadRequest, t)
	verifyAuthBasicAPIErrorResponse(shortNameResp.JSON400, t)

	// Password below minLength: 10 — must be rejected.
	shortPasswordResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: username(), Password: "short"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(shortPasswordResp.StatusCode(), http.StatusBadRequest, t)
	verifyAuthBasicAPIErrorResponse(shortPasswordResp.JSON400, t)

	// Password above maxLength: 40 — must be rejected.
	longPasswordResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: username(), Password: "this-password-is-way-too-long-for-the-schema-limits"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(longPasswordResp.StatusCode(), http.StatusBadRequest, t)
	verifyAuthBasicAPIErrorResponse(longPasswordResp.JSON400, t)
}

// TestUserChangePassword verifies the full change-password flow: a user can log in,
// change their password, and then log in again with the new password.
func TestUserChangePassword(t *testing.T) {
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

	oldPassword := "oldpassword123"
	newPassword := "newpassword456"
	name := username()

	createUserResp2, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: name, Password: oldPassword},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp2.StatusCode(), http.StatusCreated, t)
	userID2 := createUserResp2.JSON201.Id

	loginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{Username: name, Password: oldPassword},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	session := extractSession(loginResp.HTTPResponse, t)

	changeResp, err := basicAuthClient.ChangePasswordWithResponse(
		t.Context(),
		orgID,
		userID2,
		authbasicapi.ChangePasswordJSONRequestBody{OldPassword: oldPassword, Password: newPassword},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(changeResp.StatusCode(), http.StatusNoContent, t)

	// Login with new password must succeed.
	newLoginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{Username: name, Password: newPassword},
	)
	checkErr(err, t)
	verifyStatusCode(newLoginResp.StatusCode(), http.StatusNoContent, t)

	// Login with old password must now fail.
	oldLoginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{Username: name, Password: oldPassword},
	)
	checkErr(err, t)
	verifyStatusCode(oldLoginResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(oldLoginResp.JSON401, t)
}

// TestUserChangePasswordOASValidation verifies that the OAS validator rejects change-password
// requests with credentials that violate the schema length constraints.
// Note: the spec does not define a 400 response body for this endpoint, so only the
// status code is checked.
func TestUserChangePasswordOASValidation(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	// oldPassword below minLength: 10 — must be rejected before auth checks.
	shortOldPwResp, err := basicAuthClient.ChangePasswordWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.ChangePasswordJSONRequestBody{OldPassword: "short", Password: "validpassword123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(shortOldPwResp.StatusCode(), http.StatusBadRequest, t)

	// new password below minLength: 10 — must be rejected.
	shortNewPwResp, err := basicAuthClient.ChangePasswordWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.ChangePasswordJSONRequestBody{OldPassword: "validoldpassword", Password: "short"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(shortNewPwResp.StatusCode(), http.StatusBadRequest, t)
}

// TestUserNoSession verifies that every user-scoped endpoint returns 401 with a populated
// error body when called without a session header.
func TestUserNoSession(t *testing.T) {
	// CreateUser — no session.
	createResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(createResp.JSON401, t)

	// ListUsers — no session.
	listResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(listResp.JSON401, t)

	// GetUser — no session.
	getResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(getResp.JSON401, t)

	// UpdateUser — no session.
	updateResp, err := basicAuthClient.UpdateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.UpdateUserJSONRequestBody{Id: int64(alwaysUserID), Name: username()},
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(updateResp.JSON401, t)

	// DeleteUser — no session.
	deleteResp, err := basicAuthClient.DeleteUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(deleteResp.JSON401, t)

	// UpdateUserGroups — no session.
	updateGroupsResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.UpdateUserGroupsJSONRequestBody([]string{}),
	)
	checkErr(err, t)
	verifyStatusCode(updateGroupsResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(updateGroupsResp.JSON401, t)

	// GetUserGroups — no session.
	getGroupsResp, err := basicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
	)
	checkErr(err, t)
	verifyStatusCode(getGroupsResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(getGroupsResp.JSON401, t)

	// ChangePassword — no session.
	changePwResp, err := basicAuthClient.ChangePasswordWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.ChangePasswordJSONRequestBody{OldPassword: "validoldpassword", Password: "validnewpassword"},
	)
	checkErr(err, t)
	verifyStatusCode(changePwResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(changePwResp.JSON401, t)
}

// TestUserDeleteNotFound verifies deleting an already-deleted user.
func TestUserDeleteNotFound(t *testing.T) {
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

	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	// First delete succeeds.
	deleteResp, err := basicAuthClient.DeleteUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Second delete must return 404 (no body defined in spec).
	deleteAgainResp, err := basicAuthClient.DeleteUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteAgainResp.StatusCode(), http.StatusNoContent, t)
}

// TestUserUpdateNotFound verifies that attempting to update a deleted user returns 404
// (no body defined in spec).
func TestUserUpdateNotFound(t *testing.T) {
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

	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	// Delete the user first.
	deleteResp, err := basicAuthClient.DeleteUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Update the deleted user must return 404 (no body defined in spec).
	updateResp, err := basicAuthClient.UpdateUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserJSONRequestBody{Id: userID, Name: username()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNotFound, t)
}
