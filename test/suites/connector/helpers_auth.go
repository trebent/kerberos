package connector

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

func loginAndGetSessionCookie(t *testing.T) *http.Cookie {
	loginResponse, err := adminClient.LoginWithResponse(t.Context(), adminapi.LoginJSONRequestBody{
		Username: adminUser,
		Password: adminUserPassword,
	})
	checkErr(err, t)
	verifyStatusCode(loginResponse.StatusCode(), http.StatusNoContent, t)

	sessionCookie, err := extractSessionCookie(loginResponse.HTTPResponse)
	checkErr(err, t)

	return sessionCookie
}
