package lib

import (
	"net/http"
	"testing"

	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

// OrgWithSession creates a fresh organisation and returns its ID along with an
// admin session request editor for that organisation.
func OrgWithSession(t *testing.T, requestEditor RequestEditorFn) (authbasicapi.Orgid, RequestEditorFn) {
	t.Helper()
	createOrgResp, err := BasicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		authbasicapi.CreateOrganisationJSONRequestBody{Name: OrgName()},
		authbasicapi.RequestEditorFn(requestEditor),
	)
	CheckErr(err, t)
	VerifyStatusCode(createOrgResp.StatusCode(), http.StatusCreated, t)

	loginResp, err := BasicAuthClient.LoginWithResponse(
		t.Context(),
		createOrgResp.JSON201.Id,
		authbasicapi.LoginJSONRequestBody{
			Username: createOrgResp.JSON201.AdminUsername,
			Password: createOrgResp.JSON201.AdminPassword,
		},
	)
	CheckErr(err, t)
	VerifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)

	return createOrgResp.JSON201.Id, SessionCookieRequestEditor(loginResp.HTTPResponse, t)
}
