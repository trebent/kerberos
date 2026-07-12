package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
)

// TestAdminRefreshSuperuserSessionNoRefreshCookie verifies that calling the superuser refresh
// endpoint without a refresh cookie returns 401. A missing session cookie alone is not enough
// to trigger an error — only the missing refresh cookie matters here.
func TestAdminRefreshSuperuserSessionNoRefreshCookie(t *testing.T) {
	t.Parallel()
	resp, err := adminClient.RefreshSuperuserSessionWithResponse(t.Context())
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusUnauthorized, t)
	verifyAdminAPIErrorResponse(resp.JSON401, t)
}

// TestAdminRefreshSuperuserSession verifies that the superuser refresh endpoint issues a new
// session when called with only the refresh cookie (no session cookie required).
func TestAdminRefreshSuperuserSession(t *testing.T) {
	t.Parallel()
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{
			ClientId:     superUserClientID,
			ClientSecret: superUserClientSecret,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)

	// Use only the refresh cookie — deliberately omit the session cookie to prove it is not required.
	refreshEditor := refreshCookieRequestEditor(loginResp.HTTPResponse, t)

	refreshResp, err := adminClient.RefreshSuperuserSessionWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(refreshEditor),
	)
	checkErr(err, t)
	verifyStatusCode(refreshResp.StatusCode(), http.StatusNoContent, t)
}

// TestAdminRefreshSuperuserSessionForbidden verifies that a non-superuser admin refresh token
// is rejected by the superuser refresh endpoint with 403.
func TestAdminRefreshSuperuserSessionForbidden(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)

	name := username()
	const pass = "password123"
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	// Login as the regular admin user to get a non-superuser refresh token.
	loginResp, err := adminClient.LoginWithResponse(
		t.Context(),
		adminapi.LoginJSONRequestBody{Username: name, Password: pass},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)

	// Use only the refresh cookie from the regular user session.
	userRefreshEditor := refreshCookieRequestEditor(loginResp.HTTPResponse, t)

	refreshResp, err := adminClient.RefreshSuperuserSessionWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(userRefreshEditor),
	)
	checkErr(err, t)
	verifyStatusCode(refreshResp.StatusCode(), http.StatusForbidden, t)
	verifyAdminAPIErrorResponse(refreshResp.JSON403, t)
}

// TestAdminRefreshUserSessionNoRefreshCookie verifies that calling the admin user refresh
// endpoint without a refresh cookie returns 401.
func TestAdminRefreshUserSessionNoRefreshCookie(t *testing.T) {
	t.Parallel()
	resp, err := adminClient.RefreshUserSessionWithResponse(t.Context())
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusUnauthorized, t)
	verifyAdminAPIErrorResponse(resp.JSON401, t)
}

// TestAdminRefreshUserSession verifies that the admin user refresh endpoint issues a new
// session when called with only the refresh cookie (no session cookie required).
func TestAdminRefreshUserSession(t *testing.T) {
	t.Parallel()
	superRequestEditor := superLogin(t)

	name := username()
	const pass = "password123"
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	loginResp, err := adminClient.LoginWithResponse(
		t.Context(),
		adminapi.LoginJSONRequestBody{Username: name, Password: pass},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)

	// Use only the refresh cookie — deliberately omit the session cookie.
	refreshEditor := refreshCookieRequestEditor(loginResp.HTTPResponse, t)

	refreshResp, err := adminClient.RefreshUserSessionWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(refreshEditor),
	)
	checkErr(err, t)
	verifyStatusCode(refreshResp.StatusCode(), http.StatusNoContent, t)
}
