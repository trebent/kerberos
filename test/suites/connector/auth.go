package connector

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	lib "github.com/trebent/kerberos/test/lib"
)

func loginAndGetSessionCookie(t *testing.T) *http.Cookie {
	t.Helper()
	loginResponse, err := lib.AdminClient.LoginWithResponse(
		t.Context(),
		adminapi.LoginJSONRequestBody{Username: adminUser, Password: adminUserPassword},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResponse.StatusCode(), http.StatusNoContent, t)
	sessionCookie, err := lib.ExtractSessionCookie(loginResponse.HTTPResponse)
	lib.CheckErr(err, t)
	return sessionCookie
}
