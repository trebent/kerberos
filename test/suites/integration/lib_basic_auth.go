package integration

import (
	"net/http"
	"testing"

	authbasicapi "github.com/trebent/kerberos/test/integration/client/auth/basic"
)

// orgWithSession is a helper that creates a fresh organisation and returns its ID along
// with an admin session request editor for that organisation. It uses the provided superuser session to
// create the organisation.
func orgWithSession(t *testing.T, requestEditor RequestEditorFn) (authbasicapi.Orgid, RequestEditorFn) {
	t.Helper()
	createOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: orgName()},
		authbasicapi.RequestEditorFn(requestEditor),
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

	return createOrgResp.JSON201.Id, sessionCookieRequestEditor(loginResp.HTTPResponse, t)
}
