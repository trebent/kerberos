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

	t.Run("allowed origin, OPTIONS preflight", func(t *testing.T) {
		t.Parallel()
		url := fmt.Sprintf("http://%s:%d", getHost(), getConnectorPort())
		resp := options(url, t, http.Header{"Origin": []string{"http://localhost:30100"}})
		defer resp.Body.Close()
		verifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		verifyHeader(resp.Header, "Access-Control-Allow-Origin", "http://localhost:30100", t)
		verifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
		if resp.Header.Get("Access-Control-Allow-Methods") == "" {
			t.Fatal("Expected Access-Control-Allow-Methods to be set")
		}
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
