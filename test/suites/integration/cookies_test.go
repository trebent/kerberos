package integration

import (
	"fmt"
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

func TestCookiesAdmin(t *testing.T) {
	t.Run("Verify superuser cookie attributes", func(t *testing.T) {
		t.Parallel()
		resp, err := lib.AdminClient.LoginSuperuser(
			t.Context(),
			adminapi.LoginSuperuserJSONRequestBody{
				ClientId:     lib.SuperUserClientID,
				ClientSecret: lib.SuperUserClientSecret,
			},
			// No request editor to set an Origin, should pass automatically.
		)
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		validateAdminCookieAttributes(resp.Cookies(), "/api/admin/superuser/refresh", t)
	})

	t.Run("Verify admin user cookie attributes", func(t *testing.T) {
		t.Parallel()
		resp, err := lib.AdminClient.Login(
			t.Context(),
			adminapi.LoginJSONRequestBody{
				Username: alwaysAdminUser,
				Password: alwaysUserPassword,
			},
			// No request editor to set an Origin, should pass automatically.
		)
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		validateAdminCookieAttributes(resp.Cookies(), "/api/admin/refresh", t)
	})
}

func TestCookiesBasicAuth(t *testing.T) {
	t.Run("Verify basic auth cookie attributes", func(t *testing.T) {
		t.Parallel()
		loginResp, err := lib.BasicAuthClient.Login(
			t.Context(),
			authbasicapi.Orgid(alwaysOrgID),
			authbasicapi.LoginJSONRequestBody{
				Username: alwaysUser,
				Password: alwaysUserPassword,
			})
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(loginResp.StatusCode, http.StatusNoContent, t)
		var refresh, session, csrf bool
		for _, cookie := range loginResp.Cookies() {
			switch cookie.Name {
			case "session":
				session = true
				if !cookie.Secure {
					t.Errorf("Expected 'session' cookie to have Secure attribute, but it was not set")
				}
				if !cookie.HttpOnly {
					t.Errorf("Expected 'session' cookie to have HttpOnly attribute, but it was not set")
				}
				if cookie.SameSite != http.SameSiteStrictMode {
					t.Errorf("Expected 'session' cookie to have SameSite=Strict, but got %v", cookie.SameSite)
				}
				if cookie.Domain != "" {
					t.Errorf("Expected 'session' cookie to have Domain='', but got %v", cookie.Domain)
				}
				if cookie.Path != "/" {
					t.Errorf("Expected 'session' cookie to have Path='/', but got %v", cookie.Path)
				}
			case "refresh":
				refresh = true
				if !cookie.Secure {
					t.Errorf("Expected 'refresh' cookie to have Secure attribute, but it was not set")
				}
				if !cookie.HttpOnly {
					t.Errorf("Expected 'refresh' cookie to have HttpOnly attribute, but it was not set")
				}
				if cookie.SameSite != http.SameSiteStrictMode {
					t.Errorf("Expected 'refresh' cookie to have SameSite=Strict, but got %v", cookie.SameSite)
				}
				if cookie.Domain != "" {
					t.Errorf("Expected 'refresh' cookie to have Domain='', but got %v", cookie.Domain)
				}
				expectedRefreshPath := fmt.Sprintf("/api/auth/basic/organisations/%d/refresh", alwaysOrgID)
				if cookie.Path != expectedRefreshPath {
					t.Errorf("Expected 'refresh' cookie to have Path='%s', but got %v", expectedRefreshPath, cookie.Path)
				}
			case "csrf":
				csrf = true
				if !cookie.Secure {
					t.Errorf("Expected 'csrf' cookie to have Secure attribute, but it was not set")
				}
				if cookie.HttpOnly {
					t.Errorf("Expected 'csrf' cookie to NOT have HttpOnly attribute, but it was set")
				}
				if cookie.SameSite != http.SameSiteStrictMode {
					t.Errorf("Expected 'refresh' cookie to have SameSite=Strict, but got %v", cookie.SameSite)
				}
				if cookie.Domain != "" {
					t.Errorf("Expected 'refresh' cookie to have Domain='', but got %v", cookie.Domain)
				}
				if cookie.Path != "/" {
					t.Errorf("Expected 'csrf' cookie to have Path='/', but got %v", cookie.Path)
				}
			default:
				t.Log("Ignoring cookie", cookie.Name)
			}
		}

		if !(refresh && session && csrf) {
			t.Errorf("Expected to find 'refresh', 'session', and 'csrf' cookies, but did not find all of them")
		}
	})
}

func validateAdminCookieAttributes(cookies []*http.Cookie, expectedRefreshPath string, t *testing.T) {
	var refresh, session, csrf bool
	for _, cookie := range cookies {
		switch cookie.Name {
		case "session":
			session = true
			if !cookie.Secure {
				t.Errorf("Expected 'session' cookie to have Secure attribute, but it was not set")
			}
			if !cookie.HttpOnly {
				t.Errorf("Expected 'session' cookie to have HttpOnly attribute, but it was not set")
			}
			if cookie.SameSite != http.SameSiteStrictMode {
				t.Errorf("Expected 'session' cookie to have SameSite=Lax, but got %v", cookie.SameSite)
			}
			if cookie.Domain != "" {
				t.Errorf("Expected 'session' cookie to have Domain='', but got %v", cookie.Domain)
			}
			if cookie.Path != "/" {
				t.Errorf("Expected 'session' cookie to have Path='/', but got %v", cookie.Path)
			}
		case "csrf":
			csrf = true
			if !cookie.Secure {
				t.Errorf("Expected 'csrf' cookie to have Secure attribute, but it was not set")
			}
			if cookie.HttpOnly {
				t.Errorf("Expected 'csrf' cookie to NOT have HttpOnly attribute, but it was set")
			}
			if cookie.SameSite != http.SameSiteStrictMode {
				t.Errorf("Expected 'csrf' cookie to have SameSite=Lax, but got %v", cookie.SameSite)
			}
			if cookie.Domain != "" {
				t.Errorf("Expected 'csrf' cookie to have Domain='', but got %v", cookie.Domain)
			}
			if cookie.Path != "/" {
				t.Errorf("Expected 'csrf' cookie to have Path='/', but got %v", cookie.Path)
			}
		case "refresh":
			refresh = true
			if !cookie.Secure {
				t.Errorf("Expected 'refresh' cookie to have Secure attribute, but it was not set")
			}
			if !cookie.HttpOnly {
				t.Errorf("Expected 'refresh' cookie to have HttpOnly attribute, but it was not set")
			}
			if cookie.SameSite != http.SameSiteStrictMode {
				t.Errorf("Expected 'refresh' cookie to have SameSite=Lax, but got %v", cookie.SameSite)
			}
			if cookie.Domain != "" {
				t.Errorf("Expected 'refresh' cookie to have Domain='', but got %v", cookie.Domain)
			}
			if cookie.Path != expectedRefreshPath {
				t.Errorf("Expected 'refresh' cookie to have Path='%s', but got %v", expectedRefreshPath, cookie.Path)
			}
		default:
			t.Log("Ignoring cookie", cookie.Name)
		}
	}

	if !(refresh && session && csrf) {
		t.Errorf("Expected to find 'refresh', 'session', and 'csrf' cookies, but did not find all of them")
	}
}
