package connector

import (
	"fmt"
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"net/url"
	"testing"
)

func TestSession(t *testing.T) {
	t.Run("no session cookie, unauthorized", func(t *testing.T) {
		// Make a request without a session cookie to verify it's unauthorized
		httpClient := &http.Client{}
		req := &http.Request{
			Method: "GET",
			URL: &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("%s:%d", lib.GetHost(), lib.GetConnectorPort()),
			},
			Header: make(http.Header),
		}

		response, err := httpClient.Do(req)
		lib.CheckErr(err, t)
		defer response.Body.Close()

		lib.VerifyStatusCode(response.StatusCode, http.StatusUnauthorized, t)
	})

	t.Run("invalid session cookie, unauthorized", func(t *testing.T) {
		// Make a request with a random session cookie value
		httpClient := &http.Client{}
		req := &http.Request{
			Method: "GET",
			URL: &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("%s:%d", lib.GetHost(), lib.GetConnectorPort()),
			},
			Header: make(http.Header),
		}
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: "invalid-session-value",
		})

		response, err := httpClient.Do(req)
		lib.CheckErr(err, t)
		defer response.Body.Close()

		lib.VerifyStatusCode(response.StatusCode, http.StatusUnauthorized, t)
	})

	t.Run("session cookie is valid", func(t *testing.T) {
		cookie := loginAndGetSessionCookie(t)
		if cookie == nil {
			t.Fatal("Expected session cookie to be set, but it was nil")
		}

		// Make a request with the session cookie to verify it's valid
		httpClient := &http.Client{}
		req := &http.Request{
			Method: "GET",
			URL: &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("%s:%d", lib.GetHost(), lib.GetConnectorPort()),
			},
			Header: make(http.Header),
		}
		req.AddCookie(cookie)

		response, err := httpClient.Do(req)
		lib.CheckErr(err, t)
		defer response.Body.Close()

		lib.VerifyStatusCode(response.StatusCode, http.StatusOK, t)
	})
}
