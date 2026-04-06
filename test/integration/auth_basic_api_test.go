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
