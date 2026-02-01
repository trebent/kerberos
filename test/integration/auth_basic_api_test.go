package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	authadminapi "github.com/trebent/kerberos/ft/client/auth/admin"
	authbasicapi "github.com/trebent/kerberos/ft/client/auth/basic"
)

const (
	orgName        = "Org"
	groupNameStaff = "Staff"
	userName       = "Smith"
)

// Validate org., group, user, binding creation.
func TestAuthBasicAPI(t *testing.T) {
	adminLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		authadminapi.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: "client-secret"},
	)
	checkErr(err, t)
	verifyStatusCode(adminLoginResp.StatusCode(), http.StatusNoContent, t)
	adminSession := adminLoginResp.HTTPResponse.Header.Get("x-krb-session")

	orgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationRequest{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
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
	session := loginResp.HTTPResponse.Header.Get("x-krb-session")
	if session == "" {
		t.Fatal("Did not get a session header")
	}

	userResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserRequest{Name: userName},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(userResp.StatusCode(), http.StatusCreated, t)

	groupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupNameStaff},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(groupResp.StatusCode(), http.StatusCreated, t)

	bindResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userResp.JSON201.Id,
		authbasicapi.UpdateUserGroupsJSONRequestBody([]string{groupNameStaff}),
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
	containsAll([]string(*getUserGroupsResp.JSON200), []string{groupNameStaff}, t)
}

func TestAuthBasicAPIOrganisationIsolation(t *testing.T) {
	adminLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		authadminapi.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: "client-secret"},
	)
	checkErr(err, t)
	verifyStatusCode(adminLoginResp.StatusCode(), http.StatusNoContent, t)
	adminSession := adminLoginResp.HTTPResponse.Header.Get("x-krb-session")

	createOrg1, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
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
		authbasicapi.CreateOrganisationJSONRequestBody{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
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

	createGroup1Resp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupNameStaff},
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
}

func TestAuthBasicAPIOrganisationCreateDenied(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		authadminapi.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: "client-secret"},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := superLoginResp.HTTPResponse.Header.Get("x-krb-session")

	orgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
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
	session := loginResp.HTTPResponse.Header.Get("x-krb-session")
	if session == "" {
		t.Fatal("Did not get a session header")
	}

	failedOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(failedOrgResp.StatusCode(), http.StatusForbidden, t)
}

func TestAuthBasicAPISuperuser(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		authadminapi.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: "client-secret"},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := superLoginResp.HTTPResponse.Header.Get("x-krb-session")

	orgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(orgResp.StatusCode(), http.StatusCreated, t)
	orgID := orgResp.JSON201.Id

	userResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{
			Name: "testing",
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(userResp.StatusCode(), http.StatusCreated, t)
	userID := userResp.JSON201.Id

	groupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{
			Name: "testing",
		},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(groupResp.StatusCode(), http.StatusCreated, t)

	groupBindResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UserGroups{"testing"},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(groupBindResp.StatusCode(), http.StatusOK, t)
}
