package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/integration/client/auth/basic"
)

type RequestEditorFn func(ctx context.Context, req *http.Request) error

const (
	defaultHost              = "localhost"
	defaultKerberosPort      = 30000
	defaultAdminPort         = 30001
	defaultMetricsPort       = 9464
	defaultJaegerReadAPIPort = 16685

	// certDir is relative to the test working directory (test/suites/security/).
	certDir = "../../certs"

	kerberosPort = 30000
	adminPort    = 30001
	echoPort     = 15000

	superUserClientID     = "admin"
	superUserClientSecret = "secret"

	adminUser         = "security-admin"
	adminUserPassword = "security-admin-password"

	basicAuthUser     = "security-basic-auth-user"
	basicAuthPassword = "security-basic-auth-password"
)

var orgID int64

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

// adminResponsesTLSClient returns an adminapi.ClientWithResponses that verifies the server cert against
// the test CA but sends no client certificate.
func adminResponsesTLSClient(t *testing.T) *adminapi.ClientWithResponses {
	t.Helper()
	client, err := adminapi.NewClientWithResponses(
		fmt.Sprintf("https://%s:%d", getHost(), getAdminPort()),
		adminapi.WithHTTPClient(tlsClient(t)),
	)
	checkErr(err, t)
	return client
}

// basicAuthResponsesTLSClient returns an adminapi.ClientWithResponses that verifies the server cert against
// the test CA but sends no client certificate.
func basicAuthResponsesTLSClient(t *testing.T) *authbasicapi.ClientWithResponses {
	t.Helper()
	client, err := authbasicapi.NewClientWithResponses(
		fmt.Sprintf("https://%s:%d", getHost(), getAdminPort()),
		authbasicapi.WithHTTPClient(tlsClient(t)),
	)
	checkErr(err, t)
	return client
}

// tlsClient returns an http.Client that verifies the server cert against the
// test CA but sends no client certificate.
func tlsClient(t *testing.T) *http.Client {
	t.Helper()
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caPool(t),
			},
		},
	}
}

// plainClient returns an http.Client that uses plain HTTP (no TLS).
func plainClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

// caPool loads the test CA certificate into a new cert pool.
func caPool(t *testing.T) *x509.CertPool {
	t.Helper()
	pool, err := getCAPool()
	if err != nil {
		t.Fatalf("Failed to load CA pool: %v", err)
	}

	return pool
}

func getCAPool() (*x509.CertPool, error) {
	pem, err := os.ReadFile(certDir + "/ca.crt")
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("no certificates found in ca.crt")
	}
	return pool, nil
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
