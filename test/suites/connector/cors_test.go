package connector

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestCORS(t *testing.T) {
	t.Run("no origin, no CORS headers", func(t *testing.T) {
		testCORS(t, "", false)
	})

	t.Run("allowed origin, CORS headers", func(t *testing.T) {
		testCORS(t, "http://localhost:30100", true)
	})
}

func testCORS(t *testing.T, origin string, expectCORSHeaders bool) {
	t.Helper()

	httpClient := &http.Client{}
	req := &http.Request{
		Method: "GET",
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
	defer response.Body.Close()

	if expectCORSHeaders {
		verifyStatusCode(response.StatusCode, http.StatusOK, t)
		verifyHeader(response.Header, "Access-Control-Allow-Origin", origin, t)
		verifyHeader(response.Header, "Access-Control-Allow-Credentials", "true", t)
	} else {
		verifyStatusCode(response.StatusCode, http.StatusOK, t)
		verifyHeaderMissing(response.Header, "Access-Control-Allow-Origin", t)
	}
}
