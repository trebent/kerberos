package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/ft/client/admin"
	authbasicapi "github.com/trebent/kerberos/ft/client/auth/basic"
)

// Validate org., group, user, binding creation.
func TestAuthBasicAPI(t *testing.T) {
	adminLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(adminLoginResp.StatusCode(), http.StatusNoContent, t)
	adminSession := adminLoginResp.HTTPResponse.Header.Get("x-krb-session")

	orgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationRequest{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(adminSession)),
	)
	checkErr(err, t)
	verifyStatusCode(orgResp.StatusCode(), http.StatusCreated, t)
	orgID := orgResp.JSON201.Id

	loginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{
			Username: orgResp.JSON201.AdminUsername,
			Password: orgResp.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	session := extractSession(loginResp.HTTPResponse, t)
	if session == "" {
		t.Fatal("Did not get a session header")
	}

	userResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(userResp.StatusCode(), http.StatusCreated, t)

	groupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: alwaysGroupStaff},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(groupResp.StatusCode(), http.StatusCreated, t)

	bindResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userResp.JSON201.Id,
		authbasicapi.UpdateUserGroupsJSONRequestBody([]string{alwaysGroupStaff}),
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(bindResp.StatusCode(), http.StatusOK, t)

	getOrgResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getOrgResp.StatusCode(), http.StatusOK, t)
	matches(getOrgResp.JSON200.Id, orgID, t)

	getUserResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		userResp.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getUserResp.StatusCode(), http.StatusOK, t)
	matches(getUserResp.JSON200.Id, userResp.JSON201.Id, t)

	getGroupResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		orgID,
		groupResp.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getGroupResp.StatusCode(), http.StatusOK, t)
	matches(getGroupResp.JSON200.Id, groupResp.JSON201.Id, t)

	getUserGroupsResp, err := basicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		orgID,
		userResp.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getUserGroupsResp.StatusCode(), http.StatusOK, t)
	containsAll([]string(*getUserGroupsResp.JSON200), []string{alwaysGroupStaff}, t)
}

func TestAuthBasicAPIOrganisationIsolation(t *testing.T) {
	adminLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(adminLoginResp.StatusCode(), http.StatusNoContent, t)
	adminSession := adminLoginResp.HTTPResponse.Header.Get("x-krb-session")

	createOrg1, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(adminSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrg1.StatusCode(), http.StatusCreated, t)

	loginResp1, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrg1.JSON201.AdminUsername,
			Password: createOrg1.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp1.StatusCode(), http.StatusNoContent, t)
	session1 := loginResp1.HTTPResponse.Header.Get("x-krb-session")
	if session1 == "" {
		t.Fatal("Did not get a session header")
	}

	createOrg2, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(adminSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrg2.StatusCode(), http.StatusCreated, t)

	loginResp2, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		createOrg2.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrg2.JSON201.AdminUsername,
			Password: createOrg2.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp2.StatusCode(), http.StatusNoContent, t)
	session2 := loginResp2.HTTPResponse.Header.Get("x-krb-session")
	if session2 == "" {
		t.Fatal("Did not get a session header")
	}

	// Test accessing endpoints across organisations, all operations below shall fail.
	listGroupsResp, err := basicAuthClient.ListGroupsWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(listGroupsResp.StatusCode(), http.StatusForbidden, t)

	listUsersResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusForbidden, t)

	getUserResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		createOrg1.JSON201.AdminUserId,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(getUserResp.StatusCode(), http.StatusForbidden, t)

	// Create a group in org1 with session1, then try to access it with session2 which belongs to org2 - should fail.
	createGroup1Resp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateGroupJSONRequestBody{Name: alwaysGroupStaff},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session1)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroup1Resp.StatusCode(), http.StatusCreated, t)

	getGroupResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		createGroup1Resp.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(getGroupResp.StatusCode(), http.StatusForbidden, t)

	// Create a user in org1 with session1, used for the remaining isolation checks.
	createUser1Resp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session1)),
	)
	checkErr(err, t)
	verifyStatusCode(createUser1Resp.StatusCode(), http.StatusCreated, t)
	user1ID := createUser1Resp.JSON201.Id

	// All operations below target org1 resources but use session2 (org2) - all must be forbidden.

	createUserCrossResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserCrossResp.StatusCode(), http.StatusForbidden, t)

	createGroupCrossResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupCrossResp.StatusCode(), http.StatusForbidden, t)

	updateUserCrossResp, err := basicAuthClient.UpdateUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.UpdateUserJSONRequestBody{Name: username()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(updateUserCrossResp.StatusCode(), http.StatusForbidden, t)

	updateGroupCrossResp, err := basicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		createGroup1Resp.JSON201.Id,
		authbasicapi.UpdateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(updateGroupCrossResp.StatusCode(), http.StatusForbidden, t)

	deleteUserCrossResp, err := basicAuthClient.DeleteUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteUserCrossResp.StatusCode(), http.StatusForbidden, t)

	deleteGroupCrossResp, err := basicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		createGroup1Resp.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteGroupCrossResp.StatusCode(), http.StatusForbidden, t)

	updateUserGroupsCrossResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.UpdateUserGroupsJSONRequestBody([]string{}),
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(updateUserGroupsCrossResp.StatusCode(), http.StatusForbidden, t)

	getUserGroupsCrossResp, err := basicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(getUserGroupsCrossResp.StatusCode(), http.StatusForbidden, t)

	changePasswordCrossResp, err := basicAuthClient.ChangePasswordWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.ChangePasswordJSONRequestBody{
			OldPassword: "password123",
			Password:    "newpassword456",
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(changePasswordCrossResp.StatusCode(), http.StatusForbidden, t)
}

func TestAuthBasicAPIOrganisationCreateDenied(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	orgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(orgResp.StatusCode(), http.StatusCreated, t)
	orgID := orgResp.JSON201.Id

	loginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{
			Username: orgResp.JSON201.AdminUsername,
			Password: orgResp.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	session := extractSession(loginResp.HTTPResponse, t)
	if session == "" {
		t.Fatal("Did not get a session header")
	}

	failedOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(failedOrgResp.StatusCode(), http.StatusForbidden, t)
}

func TestAuthBasicAPIOASFailure(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(), adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	orgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: ""},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(orgResp.StatusCode(), http.StatusBadRequest, t)
	verifyAuthBasicAPIErrorResponse(orgResp.JSON400, t)
}

func TestAuthBasicAPIUserConflict(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{
			ClientId:     superUserClientID,
			ClientSecret: superUserClientSecret,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	session := extractSession(loginResp.HTTPResponse, t)

	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{
			Name:     alwaysUser,
			Password: alwaysUserPassword,
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusConflict, t)

	if createUserResp.JSON409 == nil {
		t.Fatal("Expected a 409 body")
	}

	if len(createUserResp.JSON409.Errors) == 0 {
		t.Fatal("Expected 1 conflict error message")
	}

	createUser2Resp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{
			Name:     username(),
			Password: alwaysUserPassword,
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createUser2Resp.StatusCode(), http.StatusCreated, t)

	updateUserResp, err := basicAuthClient.UpdateUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(createUser2Resp.JSON201.Id),
		authbasicapi.UpdateUserJSONRequestBody{
			Name: alwaysUser,
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(updateUserResp.StatusCode(), http.StatusConflict, t)

	if updateUserResp.JSON409 == nil {
		t.Fatal("Expected a 409 body")
	}

	if len(updateUserResp.JSON409.Errors) == 0 {
		t.Fatal("Expected 1 conflict error message")
	}
}

func TestAuthBasicAPIGroupConflict(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{
			ClientId:     superUserClientID,
			ClientSecret: superUserClientSecret,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	session := extractSession(loginResp.HTTPResponse, t)

	createGroupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{
			Name: alwaysGroupDev,
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupResp.StatusCode(), http.StatusConflict, t)

	if createGroupResp.JSON409 == nil {
		t.Fatal("Expected a 409 body")
	}

	if len(createGroupResp.JSON409.Errors) == 0 {
		t.Fatal("Expected 1 conflict error message")
	}

	createGroup2Resp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupJSONRequestBody{
			Name: groupName(),
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroup2Resp.StatusCode(), http.StatusCreated, t)

	updateGroupResp, err := basicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Groupid(createGroup2Resp.JSON201.Id),
		authbasicapi.UpdateGroupJSONRequestBody{
			Name: alwaysGroupDev,
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(updateGroupResp.StatusCode(), http.StatusConflict, t)

	if updateGroupResp.JSON409 == nil {
		t.Fatal("Expected a 409 body")
	}

	if len(updateGroupResp.JSON409.Errors) == 0 {
		t.Fatal("Expected 1 conflict error message")
	}
}

// Validate organisation create, update, and delete lifecycle.
func TestAuthBasicAPILifecycleOrganisation(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	session := extractSession(loginResp.HTTPResponse, t)

	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)
	orgID := createOrgResp.JSON201.Id

	getOrgResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getOrgResp.StatusCode(), http.StatusOK, t)
	matches(getOrgResp.JSON200.Name, createOrgResp.JSON201.Name, t)

	updatedName := orgName()
	updateOrgResp, err := basicAuthClient.UpdateOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.UpdateOrganisationJSONRequestBody{Name: updatedName},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(updateOrgResp.StatusCode(), http.StatusOK, t)
	matches(updateOrgResp.JSON200.Name, updatedName, t)

	getUpdatedOrgResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getUpdatedOrgResp.StatusCode(), http.StatusOK, t)
	matches(getUpdatedOrgResp.JSON200.Name, updatedName, t)

	deleteOrgResp, err := basicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteOrgResp.StatusCode(), http.StatusNoContent, t)

	getDeletedOrgResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getDeletedOrgResp.StatusCode(), http.StatusNotFound, t)
}

// Validate user create, update, and delete lifecycle.
func TestAuthBasicAPILifecycleUser(t *testing.T) {
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

	orgLoginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: createOrgResp.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(orgLoginResp.StatusCode(), http.StatusNoContent, t)
	session := extractSession(orgLoginResp.HTTPResponse, t)

	initialName := username()
	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: initialName, Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	getUserResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getUserResp.StatusCode(), http.StatusOK, t)
	matches(getUserResp.JSON200.Name, initialName, t)

	updatedName := username()
	updateUserResp, err := basicAuthClient.UpdateUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserJSONRequestBody{Name: updatedName},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(updateUserResp.StatusCode(), http.StatusOK, t)
	matches(updateUserResp.JSON200.Name, updatedName, t)

	getUpdatedUserResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getUpdatedUserResp.StatusCode(), http.StatusOK, t)
	matches(getUpdatedUserResp.JSON200.Name, updatedName, t)

	deleteUserResp, err := basicAuthClient.DeleteUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteUserResp.StatusCode(), http.StatusNoContent, t)

	getDeletedUserResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getDeletedUserResp.StatusCode(), http.StatusNotFound, t)
}

// Validate group create, update, and delete lifecycle.
func TestAuthBasicAPILifecycleGroup(t *testing.T) {
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

	orgLoginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: createOrgResp.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(orgLoginResp.StatusCode(), http.StatusNoContent, t)
	session := extractSession(orgLoginResp.HTTPResponse, t)

	initialName := groupName()
	createGroupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: initialName},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)
	groupID := createGroupResp.JSON201.Id

	getGroupResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getGroupResp.StatusCode(), http.StatusOK, t)
	matches(getGroupResp.JSON200.Name, initialName, t)

	updatedName := groupName()
	updateGroupResp, err := basicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.UpdateGroupJSONRequestBody{Name: updatedName},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(updateGroupResp.StatusCode(), http.StatusOK, t)
	matches(updateGroupResp.JSON200.Name, updatedName, t)

	getUpdatedGroupResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getUpdatedGroupResp.StatusCode(), http.StatusOK, t)
	matches(getUpdatedGroupResp.JSON200.Name, updatedName, t)

	deleteGroupResp, err := basicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteGroupResp.StatusCode(), http.StatusNoContent, t)

	getDeletedGroupResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getDeletedGroupResp.StatusCode(), http.StatusNotFound, t)
}

// Validate that a user session is invalidated after logout.
func TestAuthBasicAPILogout(t *testing.T) {
	loginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.LoginJSONRequestBody{
			Username: alwaysUser,
			Password: alwaysUserPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	session := extractSession(loginResp.HTTPResponse, t)

	// Verify the session is valid by accessing a protected endpoint.
	getOrgResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getOrgResp.StatusCode(), http.StatusOK, t)

	logoutResp, err := basicAuthClient.LogoutWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(logoutResp.StatusCode(), http.StatusNoContent, t)

	// Verify the old session is truly invalidated.
	getOrgAfterLogoutResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getOrgAfterLogoutResp.StatusCode(), http.StatusUnauthorized, t)
}

// Validate that a user can change their password and the old password is thereafter rejected.
func TestAuthBasicAPIChangePassword(t *testing.T) {
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

	orgLoginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: createOrgResp.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(orgLoginResp.StatusCode(), http.StatusNoContent, t)
	orgSession := extractSession(orgLoginResp.HTTPResponse, t)

	const oldPassword = "oldpassword123"
	const newPassword = "newpassword456"
	newUsername := username()
	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: newUsername, Password: oldPassword},
		authbasicapi.RequestEditorFn(requestEditorSessionID(orgSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	userLoginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{
			Username: newUsername,
			Password: oldPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(userLoginResp.StatusCode(), http.StatusNoContent, t)
	userSession := extractSession(userLoginResp.HTTPResponse, t)

	changePwdResp, err := basicAuthClient.ChangePasswordWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.ChangePasswordJSONRequestBody{
			OldPassword: oldPassword,
			Password:    newPassword,
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(userSession)),
	)
	checkErr(err, t)
	verifyStatusCode(changePwdResp.StatusCode(), http.StatusNoContent, t)

	// Login with the old password should now fail.
	oldPwdLoginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{
			Username: newUsername,
			Password: oldPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(oldPwdLoginResp.StatusCode(), http.StatusUnauthorized, t)

	// Login with the new password should succeed.
	newPwdLoginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{
			Username: newUsername,
			Password: newPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(newPwdLoginResp.StatusCode(), http.StatusNoContent, t)
}

func TestAuthBasicAPIOrgConflict(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{
			ClientId:     superUserClientID,
			ClientSecret: superUserClientSecret,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	session := extractSession(loginResp.HTTPResponse, t)

	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{
			Name: alwaysOrg,
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusConflict, t)

	if createOrgResp.JSON409 == nil {
		t.Fatal("Expected a 409 body")
	}

	if len(createOrgResp.JSON409.Errors) == 0 {
		t.Fatal("Expected 1 conflict error message")
	}

	createOrg2Resp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{
			Name: orgName(),
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrg2Resp.StatusCode(), http.StatusCreated, t)

	updateOrgResp, err := basicAuthClient.UpdateOrganisationWithResponse(
		t.Context(),
		authbasicapi.Orgid(createOrg2Resp.JSON201.Id),
		authbasicapi.UpdateOrganisationJSONRequestBody{
			Name: alwaysOrg,
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(updateOrgResp.StatusCode(), http.StatusConflict, t)

	if updateOrgResp.JSON409 == nil {
		t.Fatal("Expected a 409 body")
	}

	if len(updateOrgResp.JSON409.Errors) == 0 {
		t.Fatal("Expected 1 conflict error message")
	}
}
