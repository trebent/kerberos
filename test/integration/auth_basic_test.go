package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/trebent/kerberos/ft/admin"
	"github.com/trebent/kerberos/ft/basicauth"
)

func TestAuthBasic(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		admin.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: "client-secret"},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := superLoginResp.HTTPResponse.Header.Get("x-krb-session")

	orgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		basicauth.CreateOrganisationRequest{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
		requestEditorSessionID(superSession),
	)
	checkErr(err, t)
	verifyStatusCode(orgResp.StatusCode(), http.StatusCreated, t)
	orgID := orgResp.JSON201.Id

	userResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		basicauth.CreateUserRequest{Name: "username", Password: "password"},
		requestEditorSessionID(superSession),
	)
	checkErr(err, t)
	verifyStatusCode(userResp.StatusCode(), http.StatusCreated, t)
	userID := userResp.JSON201.Id

	loginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		orgID,
		basicauth.LoginJSONRequestBody{
			Username: "username",
			Password: "password",
		},
		requestEditorSessionID(superSession),
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	session := loginResp.HTTPResponse.Header.Get("x-krb-session")

	response := get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/hello", getHost(), getPort()),
		t,
		map[string][]string{"x-krb-session": {session}},
	)

	echoResponse := verifyGWResponse(response, http.StatusOK, t)
	for key, values := range echoResponse.Headers {
		if key == "x-krb-org" {
			if values[0] != fmt.Sprintf("%d", orgID) {
				t.Fatalf("Expected org ID to be %d, was %s", orgID, values[0])
			}
		}

		if key == "x-krb-user" {
			if values[0] != fmt.Sprintf("%d", userID) {
				t.Fatalf("Expected user id to be %d, was %s", userID, values[0])
			}
		}
	}
}

func TestAuthBasicUnauthenticated(t *testing.T) {
	response := get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/hello", getHost(), getPort()),
		t,
	)

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected unauthorized, got status %d", response.StatusCode)
	}
}
