package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

func TestAdminLoginSuperuser(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	t.Log("Logging the superuser out")
	superLogoutResp, err := lib.AdminClient.LogoutSuperuserWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(superLogoutResp.StatusCode(), http.StatusNoContent, t)

	t.Log("Running a GET flow request with the old session to verify it is invalidated")
	// Verify the old session is truly invalidated by attempting to access a protected endpoint with it.
	getFlowResp, err := lib.AdminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getFlowResp.StatusCode(), http.StatusUnauthorized, t)
}

func TestAdminLoginSuperuserFailure(t *testing.T) {
	t.Parallel()
	superLoginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: lib.SuperUserClientID, ClientSecret: "not-correct"},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(superLoginResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAdminAPIErrorResponse(superLoginResp.JSON401, t)
}

func TestAdminSuperuserLoginOASValidation(t *testing.T) {
	t.Parallel()
	badSuperLoginResp, err := lib.AdminClient.LoginSuperuserWithResponse(t.Context(), adminapi.LoginSuperuserJSONRequestBody{})
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(badSuperLoginResp.StatusCode(), http.StatusBadRequest, t)
	lib.VerifyAdminAPIErrorResponse(badSuperLoginResp.JSON400, t)
}

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

// TestAdminUserLoginLogout verifies that an admin user can log in, access protected endpoints,
// log out, and that their session is invalidated afterwards.
func TestAdminUserLoginLogout(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	const pass = "loginpassword123"
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	adminRequestEditor := lib.AdminUserLogin(t, name, pass)

	// GetPermissions is accessible to any authenticated admin user (no specific permission required).
	getPermsResp, err := lib.AdminClient.GetPermissionsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getPermsResp.StatusCode(), http.StatusOK, t)

	logoutResp, err := lib.AdminClient.LogoutWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(logoutResp.StatusCode(), http.StatusNoContent, t)

	getPermsResp, err = lib.AdminClient.GetPermissionsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getPermsResp.StatusCode(), http.StatusUnauthorized, t)
}

// TestAdminUserLoginFailure verifies that login with incorrect credentials returns 401.
func TestAdminUserLoginFailure(t *testing.T) {
	t.Parallel()
	loginResp, err := lib.AdminClient.LoginWithResponse(
		t.Context(),
		adminapi.LoginJSONRequestBody{Username: "no-such-user", Password: "wrong"},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAdminAPIErrorResponse(loginResp.JSON401, t)
}

// TestAdminUserChangePassword verifies that an admin user can change their password,
// that the old credentials are rejected, and that the new credentials work.
func TestAdminUserChangePassword(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	const oldPass = "oldpassword123"
	const newPass = "newpassword456"

	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: oldPass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	adminRequestEditor := lib.AdminUserLogin(t, name, oldPass)

	changeResp, err := lib.AdminClient.ChangeUserPasswordWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.ChangeUserPasswordJSONRequestBody{OldPassword: oldPass, NewPassword: newPass},
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(changeResp.StatusCode(), http.StatusNoContent, t)

	oldLoginResp, err := lib.AdminClient.LoginWithResponse(
		t.Context(),
		adminapi.LoginJSONRequestBody{Username: name, Password: oldPass},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(oldLoginResp.StatusCode(), http.StatusUnauthorized, t)

	_ = lib.AdminUserLogin(t, name, newPass)
}

// TestAdminUserChangePasswordWrongOld verifies that providing the wrong old password is rejected.
func TestAdminUserChangePasswordWrongOld(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	const pass = "correctpassword123"

	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	changeResp, err := lib.AdminClient.ChangeUserPasswordWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.ChangeUserPasswordJSONRequestBody{OldPassword: "wrong-old-pass", NewPassword: "newpass"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(changeResp.StatusCode(), http.StatusBadRequest, t)
	lib.VerifyAdminAPIErrorResponse(changeResp.JSON400, t)
}

// TestAdminMeNormalUser verifies that a normal admin user receives their own user info from /me.
func TestAdminMeNormalUser(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	const pass = "mepassword123"
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	userRequestEditor := lib.AdminUserLogin(t, name, pass)

	meResp, err := lib.AdminClient.GetMeWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(userRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(meResp.StatusCode(), http.StatusOK, t)

	if meResp.JSON200 == nil {
		t.Fatal("Expected non-nil JSON200 body")
	}
	if meResp.JSON200.IsSuperuser {
		t.Fatal("Expected isSuperuser=false for normal admin user")
	}
	if meResp.JSON200.User == nil {
		t.Fatal("Expected non-nil user field for normal admin user")
	}
	lib.Matches(meResp.JSON200.User.Username, name, t)
	lib.Matches(meResp.JSON200.User.Id, createResp.JSON201.Id, t)
}

// TestAdminMeSuperuser verifies that the superuser receives isSuperuser=true and no user field from /me.
func TestAdminMeSuperuser(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	meResp, err := lib.AdminClient.GetMeWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(meResp.StatusCode(), http.StatusOK, t)

	if meResp.JSON200 == nil {
		t.Fatal("Expected non-nil JSON200 body")
	}
	if !meResp.JSON200.IsSuperuser {
		t.Fatal("Expected isSuperuser=true for superuser")
	}
	if meResp.JSON200.User != nil {
		t.Fatal("Expected nil user field for superuser")
	}
}

// TestAdminMeUnauthenticated verifies that calling /me without a session returns 401.
func TestAdminMeUnauthenticated(t *testing.T) {
	t.Parallel()
	meResp, err := lib.AdminClient.GetMeWithResponse(t.Context())
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(meResp.StatusCode(), http.StatusUnauthorized, t)
}
