package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
)

// mustGetAdminUserID fetches the admin user list and returns the ID of the user with the given username.
func mustGetAdminUserID(t *testing.T, superSession, name string) int {
	t.Helper()
	resp, err := adminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusOK, t)
	for _, u := range *resp.JSON200 {
		if u.Username == name {
			return u.Id
		}
	}
	t.Fatalf("admin user %q not found in list", name)
	return 0
}

// TestAdminUserCreate verifies that a new admin user can be created via a superuser session.
func TestAdminUserCreate(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: username(), Password: "password123"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
}

// TestAdminUserCreateConflict verifies that creating a duplicate admin username is rejected.
func TestAdminUserCreateConflict(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := username()
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	dupResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "other-password"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(dupResp.StatusCode(), http.StatusConflict, t)
	verifyAdminAPIErrorResponse(dupResp.JSON409, t)
}

// TestAdminUserList verifies that a newly created admin user appears in the list response.
func TestAdminUserList(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := username()
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	listResp, err := adminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusOK, t)
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
	superSession := superLogin(t)

	name := username()
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	getResp, err := adminClient.GetUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	matches(getResp.JSON200.Username, name, t)
	matches(getResp.JSON200.Id, createResp.JSON201.Id, t)
}

// TestAdminUserGetNotFound verifies that fetching a non-existent admin user returns 404.
func TestAdminUserGetNotFound(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	getResp, err := adminClient.GetUserWithResponse(
		t.Context(),
		999999999,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
	verifyAdminAPIErrorResponse(getResp.JSON404, t)
}

// TestAdminUserUpdate verifies that an admin user's username can be updated.
func TestAdminUserUpdate(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := username()
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	newName := username()
	updateResp, err := adminClient.UpdateUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.UpdateUserJSONRequestBody{Username: newName},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := adminClient.GetUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	matches(getResp.JSON200.Username, newName, t)
}

// TestAdminUserUpdateConflict verifies that updating an admin user's username to an existing username returns a conflict.
func TestAdminUserUpdateConflict(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := username()
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	name2 := username()
	createResp2, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name2, Password: "password123"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp2.StatusCode(), http.StatusCreated, t)

	updateResp, err := adminClient.UpdateUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.UpdateUserJSONRequestBody{Username: name2},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusConflict, t)
}

// TestAdminUserDelete verifies that an admin user can be deleted and is no longer retrievable.
func TestAdminUserDelete(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := username()
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	deleteResp, err := adminClient.DeleteUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := adminClient.GetUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestAdminUserDeleteNotFound verifies that deleting a non-existent admin user returns 404.
func TestAdminUserDeleteNotFound(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	deleteResp, err := adminClient.DeleteUserWithResponse(
		t.Context(),
		999999999,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNotFound, t)
	verifyAdminAPIErrorResponse(deleteResp.JSON404, t)
}

// TestAdminUserLoginLogout verifies that an admin user can log in, access protected endpoints,
// log out, and that their session is invalidated afterwards.
func TestAdminUserLoginLogout(t *testing.T) {
	superSession := superLogin(t)

	name := username()
	const pass = "loginpassword123"
	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	adminSession := adminUserLogin(t, name, pass)

	// GetPermissions is accessible to any authenticated admin user (no specific permission required).
	getPermsResp, err := adminClient.GetPermissionsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(adminSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getPermsResp.StatusCode(), http.StatusOK, t)

	logoutResp, err := adminClient.LogoutWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(adminSession)),
	)
	checkErr(err, t)
	verifyStatusCode(logoutResp.StatusCode(), http.StatusNoContent, t)

	getPermsResp, err = adminClient.GetPermissionsWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(adminSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getPermsResp.StatusCode(), http.StatusUnauthorized, t)
}

// TestAdminUserLoginFailure verifies that login with incorrect credentials returns 401.
func TestAdminUserLoginFailure(t *testing.T) {
	t.Parallel()
	loginResp, err := adminClient.LoginWithResponse(
		t.Context(),
		adminapi.LoginJSONRequestBody{Username: "no-such-user", Password: "wrong"},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAdminAPIErrorResponse(loginResp.JSON401, t)
}

// TestAdminUserChangePassword verifies that an admin user can change their password,
// that the old credentials are rejected, and that the new credentials work.
func TestAdminUserChangePassword(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := username()
	const oldPass = "oldpassword123"
	const newPass = "newpassword456"

	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: oldPass},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	session := adminUserLogin(t, name, oldPass)

	changeResp, err := adminClient.ChangeUserPasswordWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.ChangeUserPasswordJSONRequestBody{OldPassword: oldPass, NewPassword: newPass},
		adminapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(changeResp.StatusCode(), http.StatusNoContent, t)

	oldLoginResp, err := adminClient.LoginWithResponse(
		t.Context(),
		adminapi.LoginJSONRequestBody{Username: name, Password: oldPass},
	)
	checkErr(err, t)
	verifyStatusCode(oldLoginResp.StatusCode(), http.StatusUnauthorized, t)

	_ = adminUserLogin(t, name, newPass)
}

// TestAdminUserChangePasswordWrongOld verifies that providing the wrong old password is rejected.
func TestAdminUserChangePasswordWrongOld(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := username()
	const pass = "correctpassword123"

	createResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	changeResp, err := adminClient.ChangeUserPasswordWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.ChangeUserPasswordJSONRequestBody{OldPassword: "wrong-old-pass", NewPassword: "newpass"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(changeResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAdminAPIErrorResponse(changeResp.JSON401, t)
}
