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
func TestFullSetup(t *testing.T) {
	orgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(), basicauth.Organisation{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
	)
	checkErr(err, t)
	verifyStatusCode(orgResp.StatusCode(), http.StatusCreated, t)
	orgID := orgResp.JSON201.Id

	loginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(), basicauth.LoginJSONRequestBody{
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
		basicauth.CreateUserRequest{Name: fmt.Sprintf("%s-%s", userName, time.Now().String())},
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
		*userResp.JSON201.Id,
		basicauth.UpdateUserGroupsJSONRequestBody{groupNameStaff},
		requestEditorSessionID(session),
	)
	checkErr(err, t)
	verifyStatusCode(bindResp.StatusCode(), http.StatusOK, t)
}

func requestEditorSessionID(sessionID string) basicauth.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("x-krb-session", sessionID)
		return nil
	}
}
