package integration

import (
	"context"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/integration/client/auth/basic"
)

func TestCORS_admin(t *testing.T) {
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
	t.Run("Non-browser request - accepted", func(t *testing.T) {
		t.Parallel()
	})

	t.Run("Browser request - accepted", func(t *testing.T) {
		t.Parallel()
	})
}
