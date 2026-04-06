package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/ft/client/admin"
	authbasicapi "github.com/trebent/kerberos/ft/client/auth/basic"
)

// TestAuthBasicAPIOrganisationIsolation verifies that a session from one organisation
// cannot read or mutate any resource that belongs to a different organisation.
func TestAuthBasicAPIOrganisationIsolation(t *testing.T) {
	adminLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(adminLoginResp.StatusCode(), http.StatusNoContent, t)
	adminSession := extractSession(adminLoginResp.HTTPResponse, t)

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
	session1 := extractSession(loginResp1.HTTPResponse, t)

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
	session2 := extractSession(loginResp2.HTTPResponse, t)

	// All read operations below target org1 but use session2 (org2) — all must be 403.
	listGroupsResp, err := basicAuthClient.ListGroupsWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(listGroupsResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(listGroupsResp.JSON403, t)

	listUsersResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(listUsersResp.JSON403, t)

	getUserResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		createOrg1.JSON201.AdminUserId,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(getUserResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(getUserResp.JSON403, t)

	// Create a group in org1 using session1, then verify session2 cannot access it.
	createGroup1Resp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
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
	verifyAuthBasicAPIErrorResponse(getGroupResp.JSON403, t)

	// Create a user in org1 using session1; all write operations via session2 must be 403.
	createUser1Resp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session1)),
	)
	checkErr(err, t)
	verifyStatusCode(createUser1Resp.StatusCode(), http.StatusCreated, t)
	user1ID := createUser1Resp.JSON201.Id

	createUserCrossResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserCrossResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(createUserCrossResp.JSON403, t)

	createGroupCrossResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupCrossResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(createGroupCrossResp.JSON403, t)

	updateUserCrossResp, err := basicAuthClient.UpdateUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.UpdateUserJSONRequestBody{Name: username()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(updateUserCrossResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(updateUserCrossResp.JSON403, t)

	updateGroupCrossResp, err := basicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		createGroup1Resp.JSON201.Id,
		authbasicapi.UpdateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(updateGroupCrossResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(updateGroupCrossResp.JSON403, t)

	deleteUserCrossResp, err := basicAuthClient.DeleteUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteUserCrossResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(deleteUserCrossResp.JSON403, t)

	deleteGroupCrossResp, err := basicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		createGroup1Resp.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteGroupCrossResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(deleteGroupCrossResp.JSON403, t)

	updateUserGroupsCrossResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.UpdateUserGroupsJSONRequestBody([]string{}),
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(updateUserGroupsCrossResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(updateUserGroupsCrossResp.JSON403, t)

	getUserGroupsCrossResp, err := basicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(getUserGroupsCrossResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(getUserGroupsCrossResp.JSON403, t)

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
	verifyAuthBasicAPIErrorResponse(changePasswordCrossResp.JSON403, t)
}

// TestAuthBasicAPIOrgAdminListOrganisationsForbidden verifies that a session scoped to an
// organisation cannot list organisations (superuser-only operation).
// The spec does not define a 403 body for ListOrganisations, so only the status is checked.
func TestAuthBasicAPIOrgAdminListOrganisationsForbidden(t *testing.T) {
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

	orgLoginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		createOrgResp.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: createOrgResp.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(orgLoginResp.StatusCode(), http.StatusNoContent, t)
	orgSession := extractSession(orgLoginResp.HTTPResponse, t)

	listResp, err := basicAuthClient.ListOrganisationsWithResponse(
		t.Context(),
		authbasicapi.RequestEditorFn(requestEditorSessionID(orgSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(listResp.JSON403, t)

	createResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(orgSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(createResp.JSON403, t)
}

// TestAuthBasicAPINormalUserAccessControl verifies that a non-administrator user receives
// 403 (with populated error body) for all admin-only operations, and can still successfully
// retrieve their own user record.
func TestAuthBasicAPINormalUserAccessControl(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	// Create a dedicated org for this test.
	orgID, adminSession := orgWithSession(t, superSession)

	// Create a group to use in group-level checks.
	createGroupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(adminSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)
	groupID := createGroupResp.JSON201.Id

	// Create a regular (non-admin) user.
	regularUserName := username()
	regularPassword := "regularpass1"
	createRegularResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: regularUserName, Password: regularPassword},
		authbasicapi.RequestEditorFn(requestEditorSessionID(adminSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createRegularResp.StatusCode(), http.StatusCreated, t)
	regularUserID := createRegularResp.JSON201.Id

	// Create another user to test that the regular user cannot access a different user.
	createOtherResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: username(), Password: "otherpass123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(adminSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOtherResp.StatusCode(), http.StatusCreated, t)
	otherUserID := createOtherResp.JSON201.Id

	// Log in as the regular user.
	regularLoginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{Username: regularUserName, Password: regularPassword},
	)
	checkErr(err, t)
	verifyStatusCode(regularLoginResp.StatusCode(), http.StatusNoContent, t)
	regularSession := extractSession(regularLoginResp.HTTPResponse, t)

	// --- Admin-only operations that must be denied (403 + body) ---

	// CreateUser.
	createUserDenyResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserDenyResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(createUserDenyResp.JSON403, t)

	// ListUsers.
	listUsersDenyResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersDenyResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(listUsersDenyResp.JSON403, t)

	// GetUser for a different user in the same org.
	getUserOtherDenyResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		otherUserID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getUserOtherDenyResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(getUserOtherDenyResp.JSON403, t)

	// GetOrganisation.
	getOrgDenyResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getOrgDenyResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(getOrgDenyResp.JSON403, t)

	// DeleteOrganisation.
	deleteOrgDenyResp, err := basicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteOrgDenyResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(deleteOrgDenyResp.JSON403, t)

	// CreateGroup.
	createGroupDenyResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupDenyResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(createGroupDenyResp.JSON403, t)

	// ListGroups.
	listGroupsDenyResp, err := basicAuthClient.ListGroupsWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listGroupsDenyResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(listGroupsDenyResp.JSON403, t)

	// GetGroup.
	getGroupDenyResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getGroupDenyResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(getGroupDenyResp.JSON403, t)

	// UpdateGroup.
	updateGroupDenyResp, err := basicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.UpdateGroupJSONRequestBody{Id: groupID, Name: groupName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateGroupDenyResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(updateGroupDenyResp.JSON403, t)

	// DeleteGroup.
	deleteGroupDenyResp, err := basicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteGroupDenyResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(deleteGroupDenyResp.JSON403, t)

	// UpdateUserGroups.
	updateUserGroupsDenyResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		regularUserID,
		authbasicapi.UpdateUserGroupsJSONRequestBody([]string{}),
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateUserGroupsDenyResp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(updateUserGroupsDenyResp.JSON403, t)

	// --- Operations the regular user IS allowed to perform on themselves ---

	// GetUser for own record must succeed.
	getUserSelfResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		regularUserID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(regularSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getUserSelfResp.StatusCode(), http.StatusOK, t)
	matches(getUserSelfResp.JSON200.Id, regularUserID, t)
}
