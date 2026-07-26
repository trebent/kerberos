package connector

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

// Verifies only configured whitelist entries are allowed to talk to the connector.
// The connector checks the Origin header of the request. Empty Origin passes since
// non-browser client.
func TestWhitelist(t *testing.T) {
	t.Run("non-browser client", func(t *testing.T) {
		testWhitelist(t, "", true)
	})

	t.Run("allowed origin", func(t *testing.T) {
		testWhitelist(t, "http://localhost:30100", true)
	})

	t.Run("disallowed origin", func(t *testing.T) {
		testWhitelist(t, "http://localhost:30101", false)
	})
}

func testWhitelist(t *testing.T, origin string, expectAllowed bool) {
	t.Helper()

	// Refresh client to guarantee an empty jar?
	httpClient := &http.Client{}
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", getHost(), getConnectorPort()),
		},
		Header: make(http.Header),
	}
	req.AddCookie(loginAndGetSessionCookie(t))
	if origin != "" {
		req.Header.Set("Origin", origin)
	}

	response, err := httpClient.Do(req)
	checkErr(err, t)
	_ = response.Body.Close()

	if expectAllowed {
		verifyStatusCode(response.StatusCode, http.StatusOK, t)
	} else {
		verifyStatusCode(response.StatusCode, http.StatusForbidden, t)
	}
}
