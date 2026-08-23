package lib

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

func RefreshCookieRequestEditor(response *http.Response, t *testing.T) RequestEditorFn {
	t.Helper()
	refreshCookie, err := ExtractRefreshCookie(response)
	if err != nil {
		t.Fatalf("failed to extract refresh cookie: %v", err)
	}
	return MakeRequestEditorFromCookie(refreshCookie)
}

func SessionCookieRequestEditor(response *http.Response, t *testing.T) RequestEditorFn {
	t.Helper()
	sessionCookie, err := ExtractSessionCookie(response)
	if err != nil {
		t.Fatalf("failed to extract session cookie: %v", err)
	}
	return MakeRequestEditorFromCookie(sessionCookie)
}

// SuperLogin logs in as the superuser and returns a request editor to use.
func SuperLogin(t *testing.T) RequestEditorFn {
	t.Helper()
	resp, err := AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: SuperUserClientID, ClientSecret: SuperUserClientSecret},
	)
	CheckErr(err, t)
	VerifyStatusCode(resp.StatusCode(), http.StatusNoContent, t)
	return SessionCookieRequestEditor(resp.HTTPResponse, t)
}

// AdminUserLogin logs in as a non-superuser admin and returns the request editor to use.
func AdminUserLogin(t *testing.T, name, pass string) RequestEditorFn {
	t.Helper()
	resp, err := AdminClient.LoginWithResponse(
		t.Context(),
		adminapi.LoginJSONRequestBody{Username: name, Password: pass},
	)
	CheckErr(err, t)
	VerifyStatusCode(resp.StatusCode(), http.StatusNoContent, t)
	return SessionCookieRequestEditor(resp.HTTPResponse, t)
}
