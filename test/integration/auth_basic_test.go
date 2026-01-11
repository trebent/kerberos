package integration

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/trebent/kerberos/ft/admin"
	"github.com/trebent/kerberos/ft/basicauth"
)

func TestAuthBasicCall(t *testing.T) {
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
	t.Logf("Created org with ID %d", orgID)

	userResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		basicauth.CreateUserRequest{Name: "username", Password: "password"},
		requestEditorSessionID(superSession),
	)
	checkErr(err, t)
	verifyStatusCode(userResp.StatusCode(), http.StatusCreated, t)
	userID := userResp.JSON201.Id
	t.Logf("Created user with ID %d", userID)

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
	t.Log(echoResponse)
	requestHeaders := http.Header(echoResponse.Headers)
	if requestHeaders.Get("x-krb-org") != strconv.Itoa(int(orgID)) {
		t.Fatalf("OrgID %s did not match expected %d", requestHeaders.Get("x-krb-org"), orgID)
	}
	if requestHeaders.Get("x-krb-user") != strconv.Itoa(int(userID)) {
		t.Fatalf("UserID %s did not match expected %d", requestHeaders.Get("x-krb-user"), userID)
	}
}

func TestAuthBasicUnauthenticated(t *testing.T) {
	response := get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/hello", getHost(), getPort()),
		t,
	)

	echoResponse := verifyGWResponse(response, http.StatusUnauthorized, t)
	requestHeaders := http.Header(echoResponse.Headers)
	if vals := requestHeaders.Values("x-krb-user"); len(vals) != 0 {
		t.Fatal("User ID should not have been set")
	}

	if vals := requestHeaders.Values("x-krb-org"); len(vals) != 0 {
		t.Fatal("Org ID should not have been set")
	}

	response = get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/hello", getHost(), getPort()),
		t,
		http.Header{"x-krb-session": {"fake"}},
	)

	echoResponse = verifyGWResponse(response, http.StatusUnauthorized, t)
	if _, ok := echoResponse.Headers["x-krb-user"]; ok {
		t.Fatal("User ID should not have been set")
	}
	if _, ok := echoResponse.Headers["x-krb-org"]; ok {
		t.Fatal("Org ID should not have been set")
	}
}
