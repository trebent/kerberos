package security

import (
	"context"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
)

func TestCORS_admin(t *testing.T) {
	t.Run("Non-browser request", func(t *testing.T) {
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
		verifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})

	t.Run("Browser request, valid Origin", func(t *testing.T) {
		t.Parallel()
		client := responsesTLSClient(t)
		resp, err := client.LoginSuperuser(
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

	t.Run("Browser request, invalid Origin", func(t *testing.T) {
		t.Parallel()
		client := responsesTLSClient(t)
		resp, err := client.LoginSuperuser(
			t.Context(),
			adminapi.LoginSuperuserJSONRequestBody{
				ClientId:     superUserClientID,
				ClientSecret: superUserClientSecret,
			},
			func(ctx context.Context, req *http.Request) error {
				req.Header.Set("Origin", "http://www.bad.com")
				return nil
			},
		)
		checkErr(err, t)
		verifyStatusCode(resp.StatusCode, http.StatusForbidden, t)
		verifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})
}

func TestCORS_gateway(t *testing.T) {
	t.Run("Non-browser request", func(t *testing.T) {
		t.Parallel()
	})

	t.Run("Browser request, valid Origin", func(t *testing.T) {
		t.Parallel()
	})

	t.Run("Browser request, invalid Origin", func(t *testing.T) {
		t.Parallel()
	})
}
