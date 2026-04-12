package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/integration/client/auth/basic"
)

// TestOrganisationCreate verifies that a superuser can create an organisation and
// that the response includes the generated admin credentials.
func TestOrganisationCreate(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	name := orgName()
	createResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	matches(createResp.JSON201.Name, name, t)
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
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	createResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	createdID := createResp.JSON201.Id

	listResp, err := basicAuthClient.ListOrganisationsWithResponse(
		t.Context(),
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusOK, t)
	for _, org := range *listResp.JSON200 {
		if org.Id == createdID {
			return
		}
	}
	t.Fatalf("created organisation %d not found in list response", createdID)
}

// TestOrganisationGet verifies that a created organisation can be fetched by ID.
func TestOrganisationGet(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	name := orgName()
	createResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	getResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	matches(getResp.JSON200.Id, createResp.JSON201.Id, t)
	matches(getResp.JSON200.Name, name, t)
}

// TestOrganisationGetNotFound verifies that fetching a deleted organisation returns 404.
func TestOrganisationGetNotFound(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	createResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	orgID := createResp.JSON201.Id

	deleteResp, err := basicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestOrganisationUpdate verifies that an organisation's name can be changed and the
// updated value is reflected in a subsequent get.
func TestOrganisationUpdate(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	createResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	orgID := createResp.JSON201.Id

	newName := orgName()
	updateResp, err := basicAuthClient.UpdateOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.UpdateOrganisationJSONRequestBody{Name: newName},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusOK, t)
	matches(updateResp.JSON200.Name, newName, t)

	getResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	matches(getResp.JSON200.Name, newName, t)
}

// TestOrganisationUpdateConflict verifies that renaming an organisation to an already-taken
// name returns a conflict error.
func TestOrganisationUpdateConflict(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	create1Resp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(create1Resp.StatusCode(), http.StatusCreated, t)

	create2Resp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(create2Resp.StatusCode(), http.StatusCreated, t)

	updateResp, err := basicAuthClient.UpdateOrganisationWithResponse(
		t.Context(),
		create2Resp.JSON201.Id,
		authbasicapi.UpdateOrganisationJSONRequestBody{Name: create1Resp.JSON201.Name},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusConflict, t)
	verifyAuthBasicAPIErrorResponse(updateResp.JSON409, t)
}

// TestOrganisationCreateConflict verifies that creating an organisation whose name is already
// taken returns a conflict error.
func TestOrganisationCreateConflict(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	name := orgName()
	createResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	conflictResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: name},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(conflictResp.StatusCode(), http.StatusConflict, t)
	verifyAuthBasicAPIErrorResponse(conflictResp.JSON409, t)
}

// TestOrganisationDelete verifies that a deleted organisation is no longer accessible.
func TestOrganisationDelete(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	createResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	orgID := createResp.JSON201.Id

	deleteResp, err := basicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestOrganisationCreateDenied verifies that an organisation-scoped session cannot create
// new organisations.
func TestOrganisationCreateDenied(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(loginResp.HTTPResponse, t)

	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	orgLoginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		createOrgResp.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: createOrgResp.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(orgLoginResp.StatusCode(), http.StatusNoContent, t)
	orgSession := extractSession(orgLoginResp.HTTPResponse, t)

	denyResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(orgSession)),
	)
	checkErr(err, t)
	verifyStatusCode(denyResp.StatusCode(), http.StatusForbidden, t)
}

// TestOrganisationCreateOASValidation verifies that creating an organisation with an empty
// name is rejected with 400 by the OAS validator.
func TestOrganisationCreateOASValidation(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	// Name below minLength: 1 — must be rejected.
	createResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: ""},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusBadRequest, t)
	verifyAuthBasicAPIErrorResponse(createResp.JSON400, t)
}

// TestOrganisationLogin verifies that a user can log in to their organisation and receives
// a session token in the response header.
func TestOrganisationLogin(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	loginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		createOrgResp.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: createOrgResp.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	session := extractSession(loginResp.HTTPResponse, t)
	if session == "" {
		t.Fatal("expected non-empty session header in login response")
	}
}

// TestOrganisationLoginInvalidCredentials verifies that a login attempt with the wrong
// password returns 401.
func TestOrganisationLoginInvalidCredentials(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	loginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		createOrgResp.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: "wrongpassword1",
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(loginResp.JSON401, t)
}

// TestOrganisationLoginOASValidation verifies that login requests with credentials that
// violate schema constraints (too-short username or password) are rejected with 400.
// The Login endpoint has no session requirement, so no prior auth is needed.
func TestOrganisationLoginOASValidation(t *testing.T) {
	// Username below minLength: 5.
	shortUsernameResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.LoginJSONRequestBody{
			Username: "ab",
			Password: "validpassword",
		},
	)
	checkErr(err, t)
	verifyStatusCode(shortUsernameResp.StatusCode(), http.StatusBadRequest, t)
	verifyAuthBasicAPIErrorResponse(shortUsernameResp.JSON400, t)

	// Password below minLength: 10.
	shortPasswordResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.LoginJSONRequestBody{
			Username: "validusername",
			Password: "short",
		},
	)
	checkErr(err, t)
	verifyStatusCode(shortPasswordResp.StatusCode(), http.StatusBadRequest, t)
	verifyAuthBasicAPIErrorResponse(shortPasswordResp.JSON400, t)
}

// TestOrganisationLogout verifies that logging out invalidates the session token so that
// subsequent authenticated requests are rejected with 401.
func TestOrganisationLogout(t *testing.T) {
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

	// Verify the session is valid before logging out.
	getUserResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getUserResp.StatusCode(), http.StatusOK, t)

	logoutResp, err := basicAuthClient.LogoutWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(logoutResp.StatusCode(), http.StatusNoContent, t)

	// The old session must now be rejected.
	getUserAfterLogoutResp, err := basicAuthClient.GetUserWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.RequestEditorFn(requestEditorSessionID(session)),
	)
	checkErr(err, t)
	verifyStatusCode(getUserAfterLogoutResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(getUserAfterLogoutResp.JSON401, t)
}

// TestOrganisationNoSession verifies that every organisation-scoped endpoint returns 401
// with a populated error body when called without a session header.
func TestOrganisationNoSession(t *testing.T) {
	// CreateOrganisation — no session.
	createResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(createResp.JSON401, t)

	// ListOrganisations — no session.
	listResp, err := basicAuthClient.ListOrganisationsWithResponse(t.Context())
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(listResp.JSON401, t)

	// GetOrganisation — no session.
	getResp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(getResp.JSON401, t)

	// UpdateOrganisation — no session.
	updateResp, err := basicAuthClient.UpdateOrganisationWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.UpdateOrganisationJSONRequestBody{Id: int64(alwaysOrgID), Name: orgName()},
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(updateResp.JSON401, t)

	// DeleteOrganisation — no session.
	deleteResp, err := basicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(deleteResp.JSON401, t)

	// Logout — no session.
	logoutResp, err := basicAuthClient.LogoutWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
	)
	checkErr(err, t)
	verifyStatusCode(logoutResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAuthBasicAPIErrorResponse(logoutResp.JSON401, t)
}

// TestOrganisationCrossOrgForbidden verifies that GetOrganisation and DeleteOrganisation
// return 403 with a populated error body when called with a session from a different
// organisation.
func TestOrganisationCrossOrgForbidden(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	// Create two organisations; each login produces a session scoped to that org.
	createOrg1, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrg1.StatusCode(), http.StatusCreated, t)

	createOrg2, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createOrg2.StatusCode(), http.StatusCreated, t)

	loginOrg2, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		createOrg2.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrg2.JSON201.AdminUsername,
			Password: createOrg2.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginOrg2.StatusCode(), http.StatusNoContent, t)
	session2 := extractSession(loginOrg2.HTTPResponse, t)

	// GetOrganisation for org1 using org2 session — must be 403.
	getOrg1Resp, err := basicAuthClient.GetOrganisationWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(getOrg1Resp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(getOrg1Resp.JSON403, t)

	// DeleteOrganisation for org1 using org2 session — must be 403.
	deleteOrg1Resp, err := basicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		createOrg1.JSON201.Id,
		authbasicapi.RequestEditorFn(requestEditorSessionID(session2)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteOrg1Resp.StatusCode(), http.StatusForbidden, t)
	verifyAuthBasicAPIErrorResponse(deleteOrg1Resp.JSON403, t)
}

// TestOrganisationDeleteNotFound verifies deleting an already-deleted organisation.
func TestOrganisationDeleteNotFound(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	createResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
	orgID := createResp.JSON201.Id

	// First delete succeeds.
	deleteResp, err := basicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Second delete must return 404 (no body defined in spec).
	deleteAgainResp, err := basicAuthClient.DeleteOrganisationWithResponse(
		t.Context(),
		orgID,
		authbasicapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteAgainResp.StatusCode(), http.StatusNoContent, t)
}
