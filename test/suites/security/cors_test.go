package security

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	lib "github.com/trebent/kerberos/test/lib"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

func TestCORSAdmin(t *testing.T) {
	t.Run("Non-browser request", func(t *testing.T) {
		t.Parallel()
		client := lib.AdminResponsesTLSClient(t, certDir)
		resp, err := client.LoginSuperuser(
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

	t.Run("Browser request, valid Origin", func(t *testing.T) {
		t.Parallel()
		client := lib.AdminResponsesTLSClient(t, certDir)
		resp, err := client.LoginSuperuser(
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

	t.Run("Browser request, invalid Origin", func(t *testing.T) {
		t.Parallel()
		client := lib.AdminResponsesTLSClient(t, certDir)
		resp, err := client.LoginSuperuser(
			t.Context(),
			adminapi.LoginSuperuserJSONRequestBody{
				ClientId:     lib.SuperUserClientID,
				ClientSecret: lib.SuperUserClientSecret,
			},
			func(ctx context.Context, req *http.Request) error {
				req.Header.Set("Origin", "http://www.bad.com")
				return nil
			},
		)
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusForbidden, t)
		lib.VerifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})
}

func TestCORSBasicAuth(t *testing.T) {
	t.Run("Browser request - denied", func(t *testing.T) {
		t.Parallel()
		client := lib.BasicAuthResponsesTLSClient(t, certDir)
		resp, err := client.Login(t.Context(), orgID, authbasicapi.LoginJSONRequestBody{
			Username: basicAuthUser,
			Password: basicAuthPassword,
		}, func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Origin", "http://www.safe.com")
			return nil
		})
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusForbidden, t)
		lib.VerifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})

	t.Run("Non-browser request - accepted", func(t *testing.T) {
		t.Parallel()
		client := lib.BasicAuthResponsesTLSClient(t, certDir)
		resp, err := client.Login(t.Context(), orgID, authbasicapi.LoginJSONRequestBody{
			Username: basicAuthUser,
			Password: basicAuthPassword,
		})
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		lib.VerifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})
}

func TestCORSGateway(t *testing.T) {
	t.Run("Non-browser request", func(t *testing.T) {
		t.Parallel()
		client := lib.TLSClient(t, certDir)

		// mtls-echo using denyAll means we should not see a returned CORS header.
		resp, err := client.Get(fmt.Sprintf("https://localhost:%d/gw/backend/mtls-echo/hi", lib.GetPort()))
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusOK, t)
		lib.VerifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})

	t.Run("Browser request, denyAll set", func(t *testing.T) {
		t.Parallel()
		client := lib.TLSClient(t, certDir)

		url, _ := url.Parse(fmt.Sprintf("https://localhost:%d/gw/backend/mtls-echo/hi", lib.GetPort()))
		req := &http.Request{
			Method: http.MethodGet,
			URL:    url,
			Header: http.Header{
				"Origin": []string{"http://www.safe.com"},
			},
		}

		// mtls-echo using denyAll means we should get denied.
		resp, err := client.Do(req)
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusForbidden, t)
		lib.VerifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})

	// tls-echo using allowedOrigins means we should see a returned CORS header for a valid Origin.
	t.Run("Browser request, valid Origin", func(t *testing.T) {
		t.Parallel()
		client := lib.TLSClient(t, certDir)

		url, _ := url.Parse(fmt.Sprintf("https://localhost:%d/gw/backend/tls-echo/hi", lib.GetPort()))
		req := &http.Request{
			Method: http.MethodGet,
			URL:    url,
			Header: http.Header{
				"Origin": []string{"http://www.safe.com"},
			},
		}

		resp, err := client.Do(req)
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusOK, t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://www.safe.com", t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
	})

	// tls-echo using allowedOrigins means we should not see a returned CORS header for an invalid Origin.
	t.Run("Browser request, invalid Origin", func(t *testing.T) {
		t.Parallel()
		client := lib.TLSClient(t, certDir)

		url, _ := url.Parse(fmt.Sprintf("https://localhost:%d/gw/backend/tls-echo/hi", lib.GetPort()))
		req := &http.Request{
			Method: http.MethodGet,
			URL:    url,
			Header: http.Header{
				"Origin": []string{"http://www.unsafe.com"},
			},
		}

		resp, err := client.Do(req)
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(resp.StatusCode, http.StatusForbidden, t)
		lib.VerifyHeaderMissing(resp.Header, "Access-Control-Allow-Origin", t)
	})
}
