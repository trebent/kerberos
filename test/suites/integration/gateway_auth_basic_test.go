package integration

import (
	"fmt"
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"strconv"
	"testing"

	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

func TestGatewayAuthBasicCall(t *testing.T) {
	loginResp, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.LoginJSONRequestBody{
			Username: alwaysUser,
			Password: alwaysUserPassword,
		},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	sessionCookie, err := lib.ExtractSessionCookie(loginResp.HTTPResponse)
	lib.CheckErr(err, t)

	response := lib.ProtectedGet(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/hi", lib.GetHost(), lib.GetPort()),
		t,
		sessionCookie,
	)

	echoResponse := lib.VerifyGWResponse(response, http.StatusOK, t)
	requestHeaders := http.Header(echoResponse.Headers)
	if requestHeaders.Get("x-krb-org") != strconv.Itoa(int(alwaysOrgID)) {
		t.Fatalf("OrgID %s did not match expected %d", requestHeaders.Get("x-krb-org"), alwaysOrgID)
	}
	if requestHeaders.Get("x-krb-user") != strconv.Itoa(int(alwaysUserID)) {
		t.Fatalf("UserID %s did not match expected %d", requestHeaders.Get("x-krb-user"), alwaysUserID)
	}
	if vals := requestHeaders.Values("x-krb-groups"); len(vals) == 0 {
		t.Fatal("Groups should have been set")
	}
}

func TestGatewayAuthBasicUnauthenticated(t *testing.T) {
	response := lib.Get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/hi", lib.GetHost(), lib.GetPort()),
		t,
	)

	echoResponse := lib.VerifyGWResponse(response, http.StatusUnauthorized, t)
	requestHeaders := http.Header(echoResponse.Headers)
	if vals := requestHeaders.Values("x-krb-user"); len(vals) != 0 {
		t.Fatal("User ID should not have been set")
	}

	if vals := requestHeaders.Values("x-krb-org"); len(vals) != 0 {
		t.Fatal("Org ID should not have been set")
	}

	response = lib.Get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/hi", lib.GetHost(), lib.GetPort()),
		t,
		http.Header{"x-krb-session": {"fake"}},
	)

	echoResponse = lib.VerifyGWResponse(response, http.StatusUnauthorized, t)
	if _, ok := echoResponse.Headers["x-krb-user"]; ok {
		t.Fatal("User ID should not have been set")
	}
	if _, ok := echoResponse.Headers["x-krb-org"]; ok {
		t.Fatal("Org ID should not have been set")
	}
}

func TestGatewayAuthBasicUnauthenticatedExempted(t *testing.T) {
	response := lib.Get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/unprotected", lib.GetHost(), lib.GetPort()),
		t,
	)

	echoResponse := lib.VerifyGWResponse(response, http.StatusOK, t)
	requestHeaders := http.Header(echoResponse.Headers)
	if vals := requestHeaders.Values("x-krb-user"); len(vals) != 0 {
		t.Fatal("User ID should not have been set")
	}

	if vals := requestHeaders.Values("x-krb-org"); len(vals) != 0 {
		t.Fatal("Org ID should not have been set")
	}

	response = lib.Get(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/unprotected/nested", lib.GetHost(), lib.GetPort()),
		t,
	)

	echoResponse = lib.VerifyGWResponse(response, http.StatusOK, t)
	requestHeaders = http.Header(echoResponse.Headers)
	if vals := requestHeaders.Values("x-krb-user"); len(vals) != 0 {
		t.Fatal("User ID should not have been set")
	}

	if vals := requestHeaders.Values("x-krb-org"); len(vals) != 0 {
		t.Fatal("Org ID should not have been set")
	}
}

func TestGatewayAuthBasicAuthorizedPleb(t *testing.T) {
	loginResp, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.LoginJSONRequestBody{
			Username: alwaysUser,
			Password: alwaysUserPassword,
		},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	sessionCookie, err := lib.ExtractSessionCookie(loginResp.HTTPResponse)
	lib.CheckErr(err, t)

	response := lib.ProtectedGet(
		fmt.Sprintf("http://%s:%d/gw/backend/protected-echo/long/hello", lib.GetHost(), lib.GetPort()),
		t,
		sessionCookie,
	)

	echoResponse := lib.VerifyGWResponse(response, http.StatusOK, t)
	requestHeaders := http.Header(echoResponse.Headers)
	if vals := requestHeaders.Values("x-krb-user"); len(vals) == 0 {
		t.Fatal("User ID should have been set")
	}

	if vals := requestHeaders.Values("x-krb-org"); len(vals) == 0 {
		t.Fatal("Org ID should have been set")
	}

	if vals := requestHeaders.Values("x-krb-groups"); len(vals) == 0 {
		t.Fatal("Groups should have been set")
	}
}
