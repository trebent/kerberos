package integration

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	authbasicapi "github.com/trebent/kerberos/ft/client/auth/basic"
)

func TestAuthBasicCall(t *testing.T) {
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

	response := get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/hi", getHost(), getPort()),
		t,
		map[string][]string{"x-krb-session": {session}},
	)

	echoResponse := verifyGWResponse(response, http.StatusOK, t)
	requestHeaders := http.Header(echoResponse.Headers)
	if requestHeaders.Get("x-krb-org") != strconv.Itoa(int(alwaysOrgID)) {
		t.Fatalf("OrgID %s did not match expected %d", requestHeaders.Get("x-krb-org"), alwaysOrgID)
	}
	if requestHeaders.Get("x-krb-user") != strconv.Itoa(int(alwaysUserID)) {
		t.Fatalf("UserID %s did not match expected %d", requestHeaders.Get("x-krb-user"), alwaysUserID)
	}
}

func TestAuthBasicUnauthenticated(t *testing.T) {
	response := get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/hi", getHost(), getPort()),
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
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/hi", getHost(), getPort()),
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

func TestAuthBasicUnauthenticatedExempted(t *testing.T) {
	response := get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/unprotected", getHost(), getPort()),
		t,
	)

	echoResponse := verifyGWResponse(response, http.StatusOK, t)
	requestHeaders := http.Header(echoResponse.Headers)
	if vals := requestHeaders.Values("x-krb-user"); len(vals) != 0 {
		t.Fatal("User ID should not have been set")
	}

	if vals := requestHeaders.Values("x-krb-org"); len(vals) != 0 {
		t.Fatal("Org ID should not have been set")
	}

	response = get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/unprotected/nested", getHost(), getPort()),
		t,
	)

	echoResponse = verifyGWResponse(response, http.StatusOK, t)
	requestHeaders = http.Header(echoResponse.Headers)
	if vals := requestHeaders.Values("x-krb-user"); len(vals) != 0 {
		t.Fatal("User ID should not have been set")
	}

	if vals := requestHeaders.Values("x-krb-org"); len(vals) != 0 {
		t.Fatal("Org ID should not have been set")
	}
}
