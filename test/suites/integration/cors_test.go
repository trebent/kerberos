package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/integration/client/auth/basic"
)

func TestCORS_admin(t *testing.T) {
	t.Run("OPTIONS preflight with Origin - CORS headers returned", func(t *testing.T) {
		t.Parallel()
		url := fmt.Sprintf("http://%s:%d/api/admin/login", getHost(), getAdminPort())
		resp := options(url, t, http.Header{"Origin": []string{"http://www.safe.com"}})
		verifyStatusCode(resp.StatusCode, http.StatusOK, t)
		verifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.safe.com", t)
		verifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
		if resp.Header.Get("Access-Control-Allow-Methods") == "" {
			t.Fatal("Expected Access-Control-Allow-Methods to be set")
		}
	})

	t.Run("Non-browser request - accepted", func(t *testing.T) {
		t.Parallel()
		resp, err := adminClient.LoginSuperuser(
			t.Context(),
			adminapi.LoginSuperuserJSONRequestBody{
				ClientId:     superUserClientID,
				ClientSecret: superUserClientSecret,
			},
			// No request editor to set an Origin, should pass automatically.
		)
		checkErr(err, t)
		verifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		verifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})

	// In the integration suite, all origins are valid.
	t.Run("Browser request - accepted", func(t *testing.T) {
		t.Parallel()
		resp, err := adminClient.LoginSuperuser(
			t.Context(),
			adminapi.LoginSuperuserJSONRequestBody{
				ClientId:     superUserClientID,
				ClientSecret: superUserClientSecret,
			},
			func(ctx context.Context, req *http.Request) error {
				req.Header.Set("Origin", "http://www.safe.com")
				return nil
			},
		)
		checkErr(err, t)
		verifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		verifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.safe.com", t)
		verifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
	})
}

func TestCORS_basicauth(t *testing.T) {
	t.Run("OPTIONS preflight with Origin - CORS headers returned", func(t *testing.T) {
		t.Parallel()
		url := fmt.Sprintf("http://%s:%d/api/auth/basic/organisations/%d/login", getHost(), getAdminPort(), alwaysOrgID)
		resp := options(url, t, http.Header{"Origin": []string{"http://www.safe.com"}})
		verifyStatusCode(resp.StatusCode, http.StatusOK, t)
		verifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.safe.com", t)
		verifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
		if resp.Header.Get("Access-Control-Allow-Methods") == "" {
			t.Fatal("Expected Access-Control-Allow-Methods to be set")
		}
	})

	t.Run("Browser request - accepted", func(t *testing.T) {
		t.Parallel()
		resp, err := basicAuthClient.Login(
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
		checkErr(err, t)
		verifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		verifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.safe.com", t)
		verifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
	})

	t.Run("Non-browser request - accepted", func(t *testing.T) {
		t.Parallel()
		resp, err := basicAuthClient.Login(
			t.Context(),
			authbasicapi.Orgid(alwaysOrgID),
			authbasicapi.LoginJSONRequestBody{
				Username: alwaysUser,
				Password: alwaysUserPassword,
			})
		checkErr(err, t)
		verifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		verifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})
}

func TestCORS_gateway(t *testing.T) {
	baseURL := fmt.Sprintf("http://localhost:%d/gw/backend/echo", getPort())

	// normal echo has allowAll, but since Origin is omitted, we should not see a returned CORS header.
	t.Run("Non-browser request - accepted", func(t *testing.T) {
		t.Parallel()
		// No Origin set
		resp := get(baseURL+"/hi", t)
		verifyStatusCode(resp.StatusCode, http.StatusOK, t)
		verifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})

	// normal echo has allowAll, expect headers
	t.Run("Browser request - accepted", func(t *testing.T) {
		t.Parallel()
		resp := get(baseURL+"/hi", t, http.Header{"Origin": []string{"http://www.something.com"}})
		verifyStatusCode(resp.StatusCode, http.StatusOK, t)
		verifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.something.com", t)
		verifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
	})

	// normal echo has allowAll, OPTIONS with Origin should return CORS headers
	t.Run("OPTIONS preflight with Origin - CORS headers returned", func(t *testing.T) {
		t.Parallel()
		resp := options(baseURL+"/hi", t, http.Header{"Origin": []string{"http://www.something.com"}})
		verifyStatusCode(resp.StatusCode, http.StatusOK, t)
		verifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.something.com", t)
		verifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
		if resp.Header.Get("Access-Control-Allow-Methods") == "" {
			t.Fatal("Expected Access-Control-Allow-Methods to be set")
		}
	})

	// Send to protected-echo since it has no CORS conf., expect no headers
	t.Run("Browser request - no configured CORS", func(t *testing.T) {
		t.Parallel()
		protectedURL := fmt.Sprintf("http://localhost:%d/gw/backend/protected-echo/unprotected", getPort())
		resp := get(protectedURL, t, http.Header{"Origin": []string{"http://www.something.com"}})
		verifyStatusCode(resp.StatusCode, http.StatusOK, t)
		verifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})
}
