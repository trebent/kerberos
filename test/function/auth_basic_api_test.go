package ft

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/trebent/kerberos/ft/basicauth"
)

//go:generate go tool oapi-codegen -config basicauth/config.yaml -o ./basicauth/clientgen.go ../../api/basic_auth.yaml

const (
	orgName        = "Org"
	groupNameStaff = "Staff"
	groupNameAdmin = "Admin"
	groupNameUser  = "User"
	userName       = "Smith"
	staticPassword = "abc123"
)

var basicAuthClient, _ = basicauth.NewClientWithResponses(
	fmt.Sprintf("http://%s:%d/api/auth/basic", getHost(), getPort()),
)

// Validate org., group, user, binding creation.
func TestAuthBasicAPIBasic(t *testing.T) {
	orgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(), basicauth.Organisation{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
	)
	checkErr(err, t)
	verifyStatusCode(orgResp.StatusCode(), http.StatusCreated, t)
	orgID := orgResp.JSON201.Id

	loginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		*orgID,
		basicauth.LoginJSONRequestBody{
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
		*orgID,
		basicauth.CreateUserRequest{Name: userName},
		requestEditorSessionID(session),
	)
	checkErr(err, t)
	verifyStatusCode(userResp.StatusCode(), http.StatusCreated, t)

	groupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		*orgID,
		basicauth.CreateGroupJSONRequestBody{Name: groupNameStaff},
		requestEditorSessionID(session),
	)
	checkErr(err, t)
	verifyStatusCode(groupResp.StatusCode(), http.StatusCreated, t)

	bindResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		*orgID,
		*userResp.JSON201.Id,
		basicauth.UpdateUserGroupsJSONRequestBody{groupNameStaff},
		requestEditorSessionID(session),
	)
	checkErr(err, t)
	verifyStatusCode(bindResp.StatusCode(), http.StatusOK, t)

	getOrgResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		*orgID,
		requestEditorSessionID(session),
	)
	checkErr(err, t)
	verifyStatusCode(getOrgResp.StatusCode(), http.StatusOK, t)
	matches(*getOrgResp.JSON200.Id, *orgID, t)

	getUserResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		*orgID,
		*userResp.JSON201.Id,
		requestEditorSessionID(session),
	)
	checkErr(err, t)
	verifyStatusCode(getUserResp.StatusCode(), http.StatusOK, t)
	matches(*getUserResp.JSON200.Id, *userResp.JSON201.Id, t)

	getGroupResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		*orgID,
		*groupResp.JSON201.Id,
		requestEditorSessionID(session),
	)
	checkErr(err, t)
	verifyStatusCode(getGroupResp.StatusCode(), http.StatusOK, t)
	matches(*getGroupResp.JSON200.Id, *groupResp.JSON201.Id, t)

	getUserGroupsResp, err := basicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		*orgID,
		*userResp.JSON201.Id,
		requestEditorSessionID(session),
	)
	checkErr(err, t)
	verifyStatusCode(getUserGroupsResp.StatusCode(), http.StatusOK, t)
	containsAll([]string(*getUserGroupsResp.JSON200), []string{groupNameStaff}, t)
}

func TestAuthBasicAPIOrganisationIsolation(t *testing.T) {
	createOrg1, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		basicauth.CreateOrganisationJSONRequestBody{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
	)
	checkErr(err, t)
	verifyStatusCode(createOrg1.StatusCode(), http.StatusCreated, t)

	loginResp1, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		*createOrg1.JSON201.Id,
		basicauth.LoginJSONRequestBody{
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
		basicauth.CreateOrganisationJSONRequestBody{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
	)
	checkErr(err, t)
	verifyStatusCode(createOrg2.StatusCode(), http.StatusCreated, t)

	loginResp2, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		*createOrg2.JSON201.Id,
		basicauth.LoginJSONRequestBody{
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
		*createOrg1.JSON201.Id,
		requestEditorSessionID(session2),
	)
	checkErr(err, t)
	verifyStatusCode(listGroupsResp.StatusCode(), http.StatusForbidden, t)

	listUsersResp, err := basicAuthClient.ListUsersWithResponse(
		t.Context(),
		*createOrg1.JSON201.Id,
		requestEditorSessionID(session2),
	)
	checkErr(err, t)
	verifyStatusCode(listUsersResp.StatusCode(), http.StatusForbidden, t)

	getUserResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		*createOrg1.JSON201.Id,
		*createOrg1.JSON201.AdminUserId,
		requestEditorSessionID(session2),
	)
	checkErr(err, t)
	verifyStatusCode(getUserResp.StatusCode(), http.StatusForbidden, t)

	createGroup1Resp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		*createOrg1.JSON201.Id,
		basicauth.CreateGroupJSONRequestBody{Name: groupNameStaff},
		requestEditorSessionID(session1),
	)
	checkErr(err, t)
	verifyStatusCode(createGroup1Resp.StatusCode(), http.StatusCreated, t)

	getGroupResp, err := basicAuthClient.GetGroupWithResponse(
		t.Context(),
		*createOrg1.JSON201.Id,
		*createGroup1Resp.JSON201.Id,
		requestEditorSessionID(session2),
	)
	checkErr(err, t)
	verifyStatusCode(getGroupResp.StatusCode(), http.StatusForbidden, t)
}

func requestEditorSessionID(sessionID string) basicauth.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("x-krb-session", sessionID)
		return nil
	}
}
