package integration

import (
	"context"
	"fmt"
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

func TestCORSAdmin(t *testing.T) {
	t.Run("OPTIONS preflight with Origin - CORS headers returned", func(t *testing.T) {
		t.Parallel()
		url := fmt.Sprintf("http://%s:%d/api/admin/login", lib.GetHost(), lib.GetAdminPort())
		resp := lib.Options(url, t, http.Header{"Origin": []string{"http://www.safe.com"}})
		lib.VerifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.safe.com", t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
		if resp.Header.Get("Access-Control-Allow-Methods") == "" {
			t.Fatal("Expected Access-Control-Allow-Methods to be set")
		}
	})

	t.Run("Non-browser request - accepted", func(t *testing.T) {
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
		lib.VerifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})

	// In the integration suite, all origins are valid.
	t.Run("Browser request - accepted", func(t *testing.T) {
		t.Parallel()
		resp, err := lib.AdminClient.LoginSuperuser(
			t.Context(),
			adminapi.LoginSuperuserJSONRequestBody{
				ClientId:     lib.SuperUserClientID,
				ClientSecret: lib.SuperUserClientSecret,
			},
			func(ctx context.Context, req *http.Request) error {
				req.Header.Set("Origin", "http://www.safe.com")
				return nil
			},
		)
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.safe.com", t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
	})
}

func TestCORSBasicAuth(t *testing.T) {
	t.Run("OPTIONS preflight with Origin - CORS headers returned", func(t *testing.T) {
		t.Parallel()
		url := fmt.Sprintf("http://%s:%d/api/auth/basic/organisations/%d/login", lib.GetHost(), lib.GetAdminPort(), alwaysOrgID)
		resp := lib.Options(url, t, http.Header{"Origin": []string{"http://www.safe.com"}})
		lib.VerifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.safe.com", t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
		if resp.Header.Get("Access-Control-Allow-Methods") == "" {
			t.Fatal("Expected Access-Control-Allow-Methods to be set")
		}
	})

	t.Run("Browser request - accepted", func(t *testing.T) {
		t.Parallel()
		resp, err := lib.BasicAuthClient.Login(
			t.Context(),
			authbasicapi.Orgid(alwaysOrgID),
			authbasicapi.LoginJSONRequestBody{
				Username: alwaysUser,
				Password: alwaysUserPassword,
			},
			func(ctx context.Context, req *http.Request) error {
				req.Header.Set("Origin", "http://www.safe.com")
				return nil
			})
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.safe.com", t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
	})

	t.Run("Non-browser request - accepted", func(t *testing.T) {
		t.Parallel()
		resp, err := lib.BasicAuthClient.Login(
			t.Context(),
			authbasicapi.Orgid(alwaysOrgID),
			authbasicapi.LoginJSONRequestBody{
				Username: alwaysUser,
				Password: alwaysUserPassword,
			})
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		lib.VerifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})
}

func TestCORSGateway(t *testing.T) {
	baseURL := fmt.Sprintf("http://localhost:%d/gw/backend/echo", lib.GetPort())

	// normal echo has allowAll, but since Origin is omitted, we should not see a returned CORS header.
	t.Run("Non-browser request - accepted", func(t *testing.T) {
		t.Parallel()
		// No Origin set
		resp := lib.Get(baseURL+"/hi", t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusOK, t)
		lib.VerifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})

	// normal echo has allowAll, expect headers
	t.Run("Browser request - accepted", func(t *testing.T) {
		t.Parallel()
		resp := lib.Get(baseURL+"/hi", t, http.Header{"Origin": []string{"http://www.something.com"}})
		lib.VerifyStatusCode(resp.StatusCode, http.StatusOK, t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.something.com", t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
	})

	// normal echo has allowAll, OPTIONS with Origin should return CORS headers
	t.Run("OPTIONS preflight with Origin - CORS headers returned", func(t *testing.T) {
		t.Parallel()
		resp := lib.Options(baseURL+"/hi", t, http.Header{"Origin": []string{"http://www.something.com"}})
		lib.VerifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.something.com", t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
		if resp.Header.Get("Access-Control-Allow-Methods") == "" {
			t.Fatal("Expected Access-Control-Allow-Methods to be set")
		}
	})

	// Send to protected-echo since it has no CORS conf., expect no headers
	t.Run("Browser request - no configured CORS", func(t *testing.T) {
		t.Parallel()
		protectedURL := fmt.Sprintf("http://localhost:%d/gw/backend/protected-echo/unprotected", lib.GetPort())
		resp := lib.Get(protectedURL, t, http.Header{"Origin": []string{"http://www.something.com"}})
		lib.VerifyStatusCode(resp.StatusCode, http.StatusOK, t)
		lib.VerifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})
}
