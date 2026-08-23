package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

// TestBasicAuthOrganisationIsolation verifies that a session from one organisation
// cannot read or mutate any resource that belongs to a different organisation.
func TestBasicAuthOrganisationIsolation(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrg1, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrg1.StatusCode(), http.StatusCreated, t)

	loginResp1, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrg1.JSON201.AdminUsername,
			Password: createOrg1.JSON201.AdminPassword,
		},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp1.StatusCode(), http.StatusNoContent, t)
	orgAdmin1RequestEditor := lib.SessionCookieRequestEditor(loginResp1.HTTPResponse, t)

	createOrg2, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrg2.StatusCode(), http.StatusCreated, t)

	loginResp2, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		createOrg2.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrg2.JSON201.AdminUsername,
			Password: createOrg2.JSON201.AdminPassword,
		},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp2.StatusCode(), http.StatusNoContent, t)
	orgAdmin2RequestEditor := lib.SessionCookieRequestEditor(loginResp2.HTTPResponse, t)

	// All read operations below target org1 but use session2 (org2) — all must be 403.
	listGroupsResp, err := lib.BasicAuthClient.ListGroupsWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listGroupsResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(listGroupsResp.JSON403, t)

	listUsersResp, err := lib.BasicAuthClient.ListUsersWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listUsersResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(listUsersResp.JSON403, t)

	getUserResp, err := lib.BasicAuthClient.GetUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		createOrg1.JSON201.AdminUserId,
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getUserResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(getUserResp.JSON403, t)

	// Create a group in org1 using session1, then verify session2 cannot access it.
	createGroup1Resp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(orgAdmin1RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroup1Resp.StatusCode(), http.StatusCreated, t)

	getGroupResp, err := lib.BasicAuthClient.GetGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		createGroup1Resp.JSON201.Id,
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getGroupResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(getGroupResp.JSON403, t)

	// Create a user in org1 using session1; all write operations via session2 must be 403.
	createUser1Resp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(orgAdmin1RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUser1Resp.StatusCode(), http.StatusCreated, t)
	user1ID := createUser1Resp.JSON201.Id

	createUserCrossResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserCrossResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(createUserCrossResp.JSON403, t)

	createGroupCrossResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupCrossResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(createGroupCrossResp.JSON403, t)

	updateUserCrossResp, err := lib.BasicAuthClient.UpdateUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.UpdateUserJSONRequestBody{Name: lib.Username()},
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateUserCrossResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateUserCrossResp.JSON403, t)

	updateGroupCrossResp, err := lib.BasicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		createGroup1Resp.JSON201.Id,
		authbasicapi.UpdateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateGroupCrossResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateGroupCrossResp.JSON403, t)

	deleteUserCrossResp, err := lib.BasicAuthClient.DeleteUserWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteUserCrossResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(deleteUserCrossResp.JSON403, t)

	deleteGroupCrossResp, err := lib.BasicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		createGroup1Resp.JSON201.Id,
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteGroupCrossResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(deleteGroupCrossResp.JSON403, t)

	updateUserGroupsCrossResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{},
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateUserGroupsCrossResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateUserGroupsCrossResp.JSON403, t)

	getUserGroupsCrossResp, err := lib.BasicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getUserGroupsCrossResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(getUserGroupsCrossResp.JSON403, t)

	changePasswordCrossResp, err := lib.BasicAuthClient.ChangePasswordWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		user1ID,
		authbasicapi.ChangePasswordJSONRequestBody{
			OldPassword: "password123",
			Password:    "newpassword456",
		},
		authbasicapi.RequestEditorFn(orgAdmin2RequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(changePasswordCrossResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(changePasswordCrossResp.JSON403, t)
}

// TestBasicAuthOrgAdminListOrganisationsForbidden verifies that a session scoped to an
// organisation cannot list organisations (superuser-only operation).
// The spec does not define a 403 body for ListOrganisations, so only the status is checked.
func TestBasicAuthOrgAdminListOrganisationsForbidden(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	orgLoginResp, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		createOrgResp.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: createOrgResp.JSON201.AdminPassword,
		},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(orgLoginResp.StatusCode(), http.StatusNoContent, t)
	orgAdminRequestEditor := lib.SessionCookieRequestEditor(orgLoginResp.HTTPResponse, t)

	listResp, err := lib.BasicAuthClient.ListOrganisationsWithResponse(
		t.Context(),
		authbasicapi.RequestEditorFn(orgAdminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(listResp.JSON403, t)

	createResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(orgAdminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(createResp.JSON403, t)
}

// TestBasicAuthNormalUserAccessControl verifies that a non-administrator user receives
// 403 (with populated error body) for all admin-only operations, and can still successfully
// retrieve their own user record.
func TestBasicAuthNormalUserAccessControl(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	// Create a dedicated org for this test.
	orgID, orgAdminRequestEditor := lib.OrgWithSession(t, superRequestEditor)

	// Create a group to use in group-level checks.
	createGroupResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(orgAdminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)
	groupID := createGroupResp.JSON201.Id

	// Create a regular (non-admin) user.
	regularUserName := lib.Username()
	regularPassword := "regularpass1"
	createRegularResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: regularUserName, Password: regularPassword},
		authbasicapi.RequestEditorFn(orgAdminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createRegularResp.StatusCode(), http.StatusCreated, t)
	regularUserID := createRegularResp.JSON201.Id

	// Create another user to test that the regular user cannot access a different user.
	createOtherResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "otherpass123"},
		authbasicapi.RequestEditorFn(orgAdminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOtherResp.StatusCode(), http.StatusCreated, t)
	otherUserID := createOtherResp.JSON201.Id

	// Log in as the regular user.
	regularLoginResp, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		authbasicapi.LoginJSONRequestBody{Username: regularUserName, Password: regularPassword},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(regularLoginResp.StatusCode(), http.StatusNoContent, t)
	orgUserRequestEditor := lib.SessionCookieRequestEditor(regularLoginResp.HTTPResponse, t)

	// --- Admin-only operations that must be denied (403 + body) ---

	// CreateUser.
	createUserDenyResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserDenyResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(createUserDenyResp.JSON403, t)

	// ListUsers.
	listUsersDenyResp, err := lib.BasicAuthClient.ListUsersWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listUsersDenyResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(listUsersDenyResp.JSON403, t)

	// GetUser for a different user in the same org.
	getUserOtherDenyResp, err := lib.BasicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		otherUserID,
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getUserOtherDenyResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(getUserOtherDenyResp.JSON403, t)

	// GetOrganisation.
	getOrgDenyResp, err := lib.BasicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getOrgDenyResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(getOrgDenyResp.JSON403, t)

	// DeleteOrganisation.
	deleteOrgDenyResp, err := lib.BasicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteOrgDenyResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(deleteOrgDenyResp.JSON403, t)

	// CreateGroup.
	createGroupDenyResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupDenyResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(createGroupDenyResp.JSON403, t)

	// ListGroups.
	listGroupsDenyResp, err := lib.BasicAuthClient.ListGroupsWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listGroupsDenyResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(listGroupsDenyResp.JSON403, t)

	// GetGroup.
	getGroupDenyResp, err := lib.BasicAuthClient.GetGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getGroupDenyResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(getGroupDenyResp.JSON403, t)

	// UpdateGroup.
	updateGroupDenyResp, err := lib.BasicAuthClient.UpdateGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.UpdateGroupJSONRequestBody{Id: groupID, Name: lib.GroupName()},
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateGroupDenyResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateGroupDenyResp.JSON403, t)

	// DeleteGroup.
	deleteGroupDenyResp, err := lib.BasicAuthClient.DeleteGroupWithResponse(
		t.Context(),
		orgID,
		groupID,
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteGroupDenyResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(deleteGroupDenyResp.JSON403, t)

	// UpdateUserGroups.
	updateUserGroupsDenyResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		regularUserID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{},
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateUserGroupsDenyResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateUserGroupsDenyResp.JSON403, t)

	// --- Operations the regular user IS allowed to perform on themselves ---

	// GetUser for own record must succeed.
	getUserSelfResp, err := lib.BasicAuthClient.GetUserWithResponse(
		t.Context(),
		orgID,
		regularUserID,
		authbasicapi.RequestEditorFn(orgUserRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getUserSelfResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getUserSelfResp.JSON200.Id, regularUserID, t)
}
