package security

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
)

func TestCookies(t *testing.T) {
	t.Run("Verify superuser cookie attributes", func(t *testing.T) {
		t.Parallel()
		client := responsesTLSClient(t)
		resp, err := client.LoginSuperuser(
			t.Context(),
			adminapi.LoginSuperuserJSONRequestBody{
				ClientId:     superUserClientID,
				ClientSecret: superUserClientSecret,
			},
			// No request editor to set an Origin, should pass automatically.
		)
		checkErr(err, t)
		verifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		cookies := resp.Cookies()
		if len(cookies) == 0 {
			t.Fatalf("Expected at least one cookie, got none")
		}
		validateCookieAttributes(cookies, t)
	})

	t.Run("Verify admin user cookie attributes", func(t *testing.T) {
		t.Parallel()
		client := responsesTLSClient(t)
		resp, err := client.Login(
			t.Context(),
			adminapi.LoginJSONRequestBody{
				Username: adminUser,
				Password: adminUserPassword,
			},
			// No request editor to set an Origin, should pass automatically.
		)
		checkErr(err, t)
		verifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		cookies := resp.Cookies()
		if len(cookies) == 0 {
			t.Fatalf("Expected at least one cookie, got none")
		}
		validateCookieAttributes(cookies, t)
	})
}

func validateCookieAttributes(cookies []*http.Cookie, t *testing.T) {
	for _, cookie := range cookies {
		switch cookie.Name {
		case "session":
			if !cookie.Secure {
				t.Errorf("Expected 'session' cookie to have Secure attribute, but it was not set")
			}
			if !cookie.HttpOnly {
				t.Errorf("Expected 'session' cookie to have HttpOnly attribute, but it was not set")
			}
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Errorf("Expected 'session' cookie to have SameSite=Lax, but got %v", cookie.SameSite)
			}
			if cookie.Domain != "krbtest.com" {
				t.Errorf("Expected 'session' cookie to have Domain=krbtest.com, but got %v", cookie.Domain)
			}
		case "csrf":
			if !cookie.Secure {
				t.Errorf("Expected 'csrf' cookie to have Secure attribute, but it was not set")
			}
			if cookie.HttpOnly {
				t.Errorf("Expected 'csrf' cookie to NOT have HttpOnly attribute, but it was set")
			}
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Errorf("Expected 'csrf' cookie to have SameSite=Lax, but got %v", cookie.SameSite)
			}
			if cookie.Domain != "krbtest.com" {
				t.Errorf("Expected 'csrf' cookie to have Domain=krbtest.com, but got %v", cookie.Domain)
			}
		case "refresh":
			if !cookie.Secure {
				t.Errorf("Expected 'refresh' cookie to have Secure attribute, but it was not set")
			}
			if !cookie.HttpOnly {
				t.Errorf("Expected 'refresh' cookie to have HttpOnly attribute, but it was not set")
			}
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Errorf("Expected 'refresh' cookie to have SameSite=Lax, but got %v", cookie.SameSite)
			}
			if cookie.Domain != "krbtest.com" {
				t.Errorf("Expected 'refresh' cookie to have Domain=krbtest.com, but got %v", cookie.Domain)
			}
		default:
			t.Log("Ignoring cookie", cookie.Name)
		}
	}
}
