package connector

import (
	"fmt"
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"net/url"
	"testing"
)

func TestCORS(t *testing.T) {
	t.Run("no origin, no CORS headers", func(t *testing.T) {
		testCORS(t, "", false)
	})

	t.Run("allowed origin, CORS headers", func(t *testing.T) {
		testCORS(t, "https://admin.trebent.test:30001", true)
	})

	t.Run("allowed origin, OPTIONS preflight", func(t *testing.T) {
		t.Parallel()
		url := fmt.Sprintf("http://%s:%d", lib.GetHost(), lib.GetConnectorPort())
		resp := lib.Options(url, t, http.Header{"Origin": []string{"https://admin.trebent.test:30001"}})
		defer resp.Body.Close()
		lib.VerifyStatusCode(resp.StatusCode, http.StatusNoContent, t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Origin", "https://admin.trebent.test:30001", t)
		lib.VerifyHeader(resp.Header, "Access-Control-Allow-Credentials", "true", t)
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
			Host:   fmt.Sprintf("%s:%d", lib.GetHost(), lib.GetConnectorPort()),
		},
		Header: make(http.Header),
	}
	req.AddCookie(loginAndGetSessionCookie(t))
	if origin != "" {
		req.Header.Set("Origin", origin)
	}

	response, err := httpClient.Do(req)
	lib.CheckErr(err, t)
	defer response.Body.Close()

	if expectCORSHeaders {
		lib.VerifyStatusCode(response.StatusCode, http.StatusOK, t)
		lib.VerifyHeader(response.Header, "Access-Control-Allow-Origin", origin, t)
		lib.VerifyHeader(response.Header, "Access-Control-Allow-Credentials", "true", t)
	} else {
		lib.VerifyStatusCode(response.StatusCode, http.StatusOK, t)
		lib.VerifyHeaderMissing(response.Header, "Access-Control-Allow-Origin", t)
	}
}
