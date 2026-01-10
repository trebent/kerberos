package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/trebent/kerberos/ft/admin"
	"github.com/trebent/kerberos/ft/basicauth"
)

func TestOrganisationCreateDenied(t *testing.T) {
	adminLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		admin.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: "client-secret"},
	)
	checkErr(err, t)
	verifyStatusCode(adminLoginResp.StatusCode(), http.StatusNoContent, t)
	adminSession := adminLoginResp.HTTPResponse.Header.Get("x-krb-session")

	orgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		basicauth.Organisation{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
		requestEditorSessionID(adminSession),
	)
	checkErr(err, t)
	verifyStatusCode(orgResp.StatusCode(), http.StatusCreated, t)
	orgID := orgResp.JSON201.Id

	loginResp, err := basicAuthClient.LoginWithResponse(
		t.Context(),
		*orgID,
		basicauth.LoginJSONRequestBody{
			Username: orgResp.JSON201.AdminUsername,
			Password: orgResp.JSON201.AdminPassword,
		},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	session := loginResp.HTTPResponse.Header.Get("x-krb-session")
	if session == "" {
		t.Fatal("Did not get a session header")
	}

	failedOrgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		t.Context(),
		basicauth.Organisation{Name: fmt.Sprintf("%s-%s", orgName, time.Now().String())},
		requestEditorSessionID(session),
	)
	checkErr(err, t)
	verifyStatusCode(failedOrgResp.StatusCode(), http.StatusForbidden, t)
}
