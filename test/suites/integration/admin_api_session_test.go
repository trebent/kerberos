package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

// TestAdminRefreshSuperuserSessionNoRefreshCookie verifies that calling the superuser refresh
// endpoint without a refresh cookie returns 401. A missing session cookie alone is not enough
// to trigger an error — only the missing refresh cookie matters here.
func TestAdminRefreshSuperuserSessionNoRefreshCookie(t *testing.T) {
	t.Parallel()
	resp, err := lib.AdminClient.RefreshSuperuserSessionWithResponse(t.Context())
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAdminAPIErrorResponse(resp.JSON401, t)
}

// TestAdminRefreshSuperuserSession verifies that the superuser refresh endpoint issues a new
// session when called with only the refresh cookie (no session cookie required).
func TestAdminRefreshSuperuserSession(t *testing.T) {
	t.Parallel()
	loginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{
			ClientId:     lib.SuperUserClientID,
			ClientSecret: lib.SuperUserClientSecret,
		},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)

	// Use only the refresh cookie — deliberately omit the session cookie to prove it is not required.
	refreshEditor := lib.RefreshCookieRequestEditor(loginResp.HTTPResponse, t)

	refreshResp, err := lib.AdminClient.RefreshSuperuserSessionWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(refreshEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(refreshResp.StatusCode(), http.StatusNoContent, t)
}

// TestAdminRefreshSuperuserSessionForbidden verifies that a non-superuser admin refresh token
// is rejected by the superuser refresh endpoint with 403.
func TestAdminRefreshSuperuserSessionForbidden(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	const pass = "password123"
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	// Login as the regular admin user to get a non-superuser refresh token.
	loginResp, err := lib.AdminClient.LoginWithResponse(
		t.Context(),
		adminapi.LoginJSONRequestBody{Username: name, Password: pass},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)

	// Use only the refresh cookie from the regular user session.
	userRefreshEditor := lib.RefreshCookieRequestEditor(loginResp.HTTPResponse, t)

	refreshResp, err := lib.AdminClient.RefreshSuperuserSessionWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(userRefreshEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(refreshResp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAdminAPIErrorResponse(refreshResp.JSON403, t)
}

// TestAdminRefreshUserSessionNoRefreshCookie verifies that calling the admin user refresh
// endpoint without a refresh cookie returns 401.
func TestAdminRefreshUserSessionNoRefreshCookie(t *testing.T) {
	t.Parallel()
	resp, err := lib.AdminClient.RefreshUserSessionWithResponse(t.Context())
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAdminAPIErrorResponse(resp.JSON401, t)
}

// TestAdminRefreshUserSession verifies that the admin user refresh endpoint issues a new
// session when called with only the refresh cookie (no session cookie required).
func TestAdminRefreshUserSession(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	const pass = "password123"
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	loginResp, err := lib.AdminClient.LoginWithResponse(
		t.Context(),
		adminapi.LoginJSONRequestBody{Username: name, Password: pass},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)

	// Use only the refresh cookie — deliberately omit the session cookie.
	refreshEditor := lib.RefreshCookieRequestEditor(loginResp.HTTPResponse, t)

	refreshResp, err := lib.AdminClient.RefreshUserSessionWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(refreshEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(refreshResp.StatusCode(), http.StatusNoContent, t)
}
