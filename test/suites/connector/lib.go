package connector

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
)

type RequestEditorFn func(ctx context.Context, req *http.Request) error

const (
	adminUser         = "connector-admin"
	adminUserPassword = "connector-admin-password"

	superUserClientID     = "admin"
	superUserClientSecret = "secret"

	defaultHost          = "localhost"
	defaultAdminPort     = 30001
	defaultConnectorPort = 30100
)

var adminClient, _ = adminapi.NewClientWithResponses(
	fmt.Sprintf("http://%s:%d", getHost(), getAdminPort()),
)

func loginAndGetSessionCookie(t *testing.T) *http.Cookie {
	loginResponse, err := adminClient.LoginWithResponse(t.Context(), adminapi.LoginJSONRequestBody{
		Username: adminUser,
		Password: adminUserPassword,
	})
	checkErr(err, t)
	verifyStatusCode(loginResponse.StatusCode(), http.StatusNoContent, t)

	sessionCookie, err := extractSessionCookie(loginResponse.HTTPResponse)
	checkErr(err, t)

	return sessionCookie
}

func getHost() string {
	hostVal, found := os.LookupEnv("KRB_FT_HOST")
	if !found {
		return defaultHost
	} else {
		return hostVal
	}
}

func getAdminPort() int {
	val, found := os.LookupEnv("KRB_FT_ADMIN_PORT")
	if !found {
		return defaultAdminPort
	}

	decoded, err := strconv.Atoi(val)
	if err != nil {
		return defaultAdminPort
	} else {
		return decoded
	}
}

func getConnectorPort() int {
	val, found := os.LookupEnv("KRB_FT_CONNECTOR_PORT")
	if !found {
		return defaultConnectorPort
	}

	decoded, err := strconv.Atoi(val)
	if err != nil {
		return defaultConnectorPort
	} else {
		return decoded
	}
}

func checkErr(err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func verifyStatusCode(in int, expected int, t *testing.T) {
	t.Helper()
	if in != expected {
		t.Fatalf("Expected status code %d, got %d", expected, in)
	}
}

func verifyHeader(headers http.Header, key string, expectedValue string, t *testing.T) {
	t.Helper()
	actualValue := headers.Get(key)
	if actualValue != expectedValue {
		t.Fatalf("Expected header %s to have value %s, got %s", key, expectedValue, actualValue)
	}
}

func verifyHeaderMissing(headers http.Header, key string, t *testing.T) {
	t.Helper()
	if headers.Get(key) != "" {
		t.Fatalf("Expected header %s to be missing, but it was present", key)
	}
}

func extractSessionCookie(resp *http.Response) (*http.Cookie, error) {
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return nil, fmt.Errorf("no cookies found in response")
	}

	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "session" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		return nil, fmt.Errorf("session cookie not found in response")
	}

	return sessionCookie, nil
}

func makeRequestEditorFromCookie(cookie *http.Cookie) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.AddCookie(cookie)
		return nil
	}
}
