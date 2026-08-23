package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

func refreshCookieRequestEditor(response *http.Response, t *testing.T) RequestEditorFn {
	t.Helper()
	refreshCookie, err := extractRefreshCookie(response)
	if err != nil {
		t.Fatalf("failed to extract refresh cookie: %v", err)
	}

	return makeRequestEditorFromCookie(refreshCookie)
}

func sessionCookieRequestEditor(response *http.Response, t *testing.T) RequestEditorFn {
	sessionCookie, err := extractSessionCookie(response)
	if err != nil {
		t.Fatalf("failed to extract session cookie: %v", err)
	}

	return makeRequestEditorFromCookie(sessionCookie)
}

// superLogin logs in as the superuser and returns a request editor to use.
func superLogin(t *testing.T) RequestEditorFn {
	t.Helper()
	resp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusNoContent, t)
	return sessionCookieRequestEditor(resp.HTTPResponse, t)
}

// adminUserLogin logs in as a non-superuser admin and returns the request editor to use.
func adminUserLogin(t *testing.T, name, pass string) RequestEditorFn {
	t.Helper()
	resp, err := adminClient.LoginWithResponse(
		t.Context(),
		adminapi.LoginJSONRequestBody{Username: name, Password: pass},
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusNoContent, t)
	return sessionCookieRequestEditor(resp.HTTPResponse, t)
}
