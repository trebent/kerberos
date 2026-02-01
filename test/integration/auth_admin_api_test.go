package integration

import (
	"net/http"
	"testing"

	authadminapi "github.com/trebent/kerberos/ft/client/auth/admin"
)

func TestAuthAdminSuperuser(t *testing.T) {
	t.Log("Logging the superuser in")
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		authadminapi.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: "client-secret"},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	t.Log("Logging the superuser out")
	superLogoutResp, err := adminClient.LogoutSuperuserWithResponse(
		t.Context(),
		authadminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatus(superLogoutResp.HTTPResponse, http.StatusNoContent, t)
}

func TestAuthAdminLoginFailure(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		authadminapi.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: "not-correct"},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusUnauthorized, t)

	// The generated client already parsed the response, so check the JSON401 response directly
	if superLoginResp.JSON401 != nil {
		t.Logf("Error response: %+v", superLoginResp.JSON401)
		if len(superLoginResp.JSON401.Errors) == 0 {
			t.Fatalf("Expected errors in response body, but got empty errors array")
		}
	} else {
		t.Fatalf("Expected JSON401 response but got nil")
	}
}

func TestAuthAdminOASFailure(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		authadminapi.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: ""},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusBadRequest, t)

	// The generated client already parsed the response, so check the JSON400 response directly
	if superLoginResp.JSON400 != nil {
		t.Logf("Error response: %+v", superLoginResp.JSON400)
		if len(superLoginResp.JSON400.Errors) == 0 {
			t.Fatalf("Expected errors in response body, but got empty errors array")
		}
	} else {
		t.Fatalf("Expected JSON401 response but got nil")
	}
}
