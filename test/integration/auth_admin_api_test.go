package integration

import (
	"net/http"
	"testing"

	authadminapi "github.com/trebent/kerberos/ft/client/auth/admin"
)

func TestAuthAdminSuperuser(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		authadminapi.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: "client-secret"},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	superLogoutResp, err := adminClient.LogoutSuperuserWithResponse(
		t.Context(),
		authadminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(superLogoutResp.StatusCode(), http.StatusNoContent, t)
}
