package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

// TestAdminUserCreate verifies that a new admin user can be created via a superuser session.
func TestAdminUserCreate(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: lib.Username(), Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
}

// TestAdminUserCreateConflict verifies that creating a duplicate admin username is rejected.
func TestAdminUserCreateConflict(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	dupResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "other-password"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(dupResp.StatusCode(), http.StatusConflict, t)
	lib.VerifyAdminAPIErrorResponse(dupResp.JSON409, t)
}

// TestAdminUserList verifies that a newly created admin user appears in the list response.
func TestAdminUserList(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	listResp, err := lib.AdminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusOK, t)
	for _, u := range *listResp.JSON200 {
		if u.Username == name {
			return
		}
	}
	t.Fatalf("admin user %q not found in list", name)
}

// TestAdminUserGet verifies that a created admin user can be fetched by ID.
func TestAdminUserGet(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	getResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Username, name, t)
	lib.Matches(getResp.JSON200.Id, createResp.JSON201.Id, t)
}

// TestAdminUserGetNotFound verifies that fetching a non-existent admin user returns 404.
func TestAdminUserGetNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	getResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		999999999,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(getResp.JSON404, t)
}

// TestAdminUserUpdate verifies that an admin user's username can be updated.
func TestAdminUserUpdate(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	newName := lib.Username()
	updateResp, err := lib.AdminClient.UpdateUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.UpdateUserJSONRequestBody{Username: newName},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Username, newName, t)
}

// TestAdminUserUpdateConflict verifies that updating an admin user's username to an existing username returns a conflict.
func TestAdminUserUpdateConflict(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	name2 := lib.Username()
	createResp2, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name2, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp2.StatusCode(), http.StatusCreated, t)

	updateResp, err := lib.AdminClient.UpdateUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.UpdateUserJSONRequestBody{Username: name2},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusConflict, t)
}

// TestAdminUserDelete verifies that an admin user can be deleted and is no longer retrievable.
func TestAdminUserDelete(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	deleteResp, err := lib.AdminClient.DeleteUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestAdminUserDeleteNotFound verifies that deleting a non-existent admin user returns 404.
func TestAdminUserDeleteNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	deleteResp, err := lib.AdminClient.DeleteUserWithResponse(
		t.Context(),
		999999999,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(deleteResp.JSON404, t)
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
