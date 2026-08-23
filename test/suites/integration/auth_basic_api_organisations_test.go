package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

// TestOrganisationCreate verifies that a superuser can create an organisation and
// that the response includes the generated admin credentials.
func TestOrganisationCreate(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.OrgName()
	createResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	lib.Matches(createResp.JSON201.Name, name, t)
	if createResp.JSON201.AdminUsername == "" {
		t.Fatal("expected non-empty admin username in create response")
	}
	if createResp.JSON201.AdminPassword == "" {
		t.Fatal("expected non-empty admin password in create response")
	}
	if createResp.JSON201.AdminUserId == 0 {
		t.Fatal("expected non-zero admin user ID in create response")
	}
}

// TestOrganisationList verifies that a newly created organisation appears in the list response.
func TestOrganisationList(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	createdID := createResp.JSON201.Id

	listResp, err := lib.BasicAuthClient.ListOrganisationsWithResponse(
		t.Context(),
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusOK, t)
	for _, org := range *listResp.JSON200 {
		if org.Id == createdID {
			return
		}
	}
	t.Fatalf("created organisation %d not found in list response", createdID)
}

// TestOrganisationGet verifies that a created organisation can be fetched by ID.
func TestOrganisationGet(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.OrgName()
	createResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	getResp, err := lib.BasicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Id, createResp.JSON201.Id, t)
	lib.Matches(getResp.JSON200.Name, name, t)
}

// TestOrganisationGetNotFound verifies that fetching a deleted organisation returns 404.
func TestOrganisationGetNotFound(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	orgID := createResp.JSON201.Id

	deleteResp, err := lib.BasicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.BasicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestOrganisationUpdate verifies that an organisation's name can be changed and the
// updated value is reflected in a subsequent get.
func TestOrganisationUpdate(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	orgID := createResp.JSON201.Id

	newName := lib.OrgName()
	updateResp, err := lib.BasicAuthClient.UpdateOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.UpdateOrganisationJSONRequestBody{Name: newName},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusOK, t)
	lib.Matches(updateResp.JSON200.Name, newName, t)

	getResp, err := lib.BasicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Name, newName, t)
}

// TestOrganisationUpdateConflict verifies that renaming an organisation to an already-taken
// name returns a conflict error.
func TestOrganisationUpdateConflict(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	create1Resp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(create1Resp.StatusCode(), http.StatusCreated, t)

	create2Resp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(create2Resp.StatusCode(), http.StatusCreated, t)

	updateResp, err := lib.BasicAuthClient.UpdateOrganisationWithResponse(
		t.Context(),
		create2Resp.JSON201.Id,
		authbasicapi.UpdateOrganisationJSONRequestBody{Name: create1Resp.JSON201.Name},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusConflict, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateResp.JSON409, t)
}

// TestOrganisationCreateConflict verifies that creating an organisation whose name is already
// taken returns a conflict error.
func TestOrganisationCreateConflict(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.OrgName()
	createResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	conflictResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(conflictResp.StatusCode(), http.StatusConflict, t)
	lib.VerifyAuthBasicAPIErrorResponse(conflictResp.JSON409, t)
}

// TestOrganisationDelete verifies that a deleted organisation is no longer accessible.
func TestOrganisationDelete(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	orgID := createResp.JSON201.Id

	deleteResp, err := lib.BasicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestOrganisationCreateDenied verifies that an organisation-scoped session cannot create
// new organisations.
func TestOrganisationCreateDenied(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	orgLoginResp, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		createOrgResp.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: createOrgResp.JSON201.AdminPassword,
		},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(orgLoginResp.StatusCode(), http.StatusNoContent, t)
	orgAdminRequestEditor := lib.SessionCookieRequestEditor(orgLoginResp.HTTPResponse, t)

	denyResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(orgAdminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(denyResp.StatusCode(), http.StatusForbidden, t)
}

// TestOrganisationCreateOASValidation verifies that creating an organisation with an empty
// name is rejected with 400 by the OAS validator.
func TestOrganisationCreateOASValidation(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	// Name below minLength: 1 — must be rejected.
	createResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: ""},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusBadRequest, t)
	lib.VerifyAuthBasicAPIErrorResponse(createResp.JSON400, t)
}

// TestOrganisationLogin verifies that a user can log in to their organisation and receives
// a session token in the response header.
func TestOrganisationLogin(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	loginResp, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		createOrgResp.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: createOrgResp.JSON201.AdminPassword,
		},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	_ = lib.SessionCookieRequestEditor(loginResp.HTTPResponse, t)
}

// TestOrganisationLoginInvalidCredentials verifies that a login attempt with the wrong
// password returns 401.
func TestOrganisationLoginInvalidCredentials(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createOrgResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	loginResp, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		createOrgResp.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: "wrongpassword1",
		},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(loginResp.JSON401, t)
}

// TestOrganisationLogout verifies that logging out invalidates the session token so that
// subsequent authenticated requests are rejected with 401.
func TestOrganisationLogout(t *testing.T) {
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
	basicRequestEditor := lib.SessionCookieRequestEditor(loginResp.HTTPResponse, t)

	// Verify the session is valid before logging out.
	getUserResp, err := lib.BasicAuthClient.GetUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.RequestEditorFn(basicRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getUserResp.StatusCode(), http.StatusOK, t)

	logoutResp, err := lib.BasicAuthClient.LogoutWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.RequestEditorFn(basicRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(logoutResp.StatusCode(), http.StatusNoContent, t)

	// The old session must now be rejected.
	getUserAfterLogoutResp, err := lib.BasicAuthClient.GetUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.RequestEditorFn(basicRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getUserAfterLogoutResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(getUserAfterLogoutResp.JSON401, t)
}

// TestOrganisationNoSession verifies that every organisation-scoped endpoint returns 401
// with a populated error body when called without a session header.
func TestOrganisationNoSession(t *testing.T) {
	// CreateOrganisation — no session.
	createResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(createResp.JSON401, t)

	// ListOrganisations — no session.
	listResp, err := lib.BasicAuthClient.ListOrganisationsWithResponse(t.Context())
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(listResp.JSON401, t)

	// GetOrganisation — no session.
	getResp, err := lib.BasicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(getResp.JSON401, t)

	// UpdateOrganisation — no session.
	updateResp, err := lib.BasicAuthClient.UpdateOrganisationWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.UpdateOrganisationJSONRequestBody{Id: int64(alwaysOrgID), Name: lib.OrgName()},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(updateResp.JSON401, t)

	// DeleteOrganisation — no session.
	deleteResp, err := lib.BasicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(deleteResp.JSON401, t)

	// Logout — no session.
	logoutResp, err := lib.BasicAuthClient.LogoutWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(logoutResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAuthBasicAPIErrorResponse(logoutResp.JSON401, t)
}

// TestOrganisationCrossOrgForbidden verifies that GetOrganisation and DeleteOrganisation
// return 403 with a populated error body when called with a session from a different
// organisation.
func TestOrganisationCrossOrgForbidden(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	// Create two organisations; each login produces a session scoped to that org.
	createOrg1, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrg1.StatusCode(), http.StatusCreated, t)

	createOrg2, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createOrg2.StatusCode(), http.StatusCreated, t)

	loginOrg2, err := lib.BasicAuthClient.LoginWithResponse(
		t.Context(),
		createOrg2.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrg2.JSON201.AdminUsername,
			Password: createOrg2.JSON201.AdminPassword,
		},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginOrg2.StatusCode(), http.StatusNoContent, t)
	orgAdminRequestEditor := lib.SessionCookieRequestEditor(loginOrg2.HTTPResponse, t)

	// GetOrganisation for org1 using org2 session — must be 403.
	getOrg1Resp, err := lib.BasicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.RequestEditorFn(orgAdminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getOrg1Resp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(getOrg1Resp.JSON403, t)

	// DeleteOrganisation for org1 using org2 session — must be 403.
	deleteOrg1Resp, err := lib.BasicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.RequestEditorFn(orgAdminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteOrg1Resp.StatusCode(), http.StatusForbidden, t)
	lib.VerifyAuthBasicAPIErrorResponse(deleteOrg1Resp.JSON403, t)
}

// TestOrganisationDeleteNotFound verifies deleting an already-deleted organisation.
func TestOrganisationDeleteNotFound(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: lib.OrgName()},
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	orgID := createResp.JSON201.Id

	// First delete succeeds.
	deleteResp, err := lib.BasicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Second delete must return 404 (no body defined in spec).
	deleteAgainResp, err := lib.BasicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteAgainResp.StatusCode(), http.StatusNoContent, t)
}
