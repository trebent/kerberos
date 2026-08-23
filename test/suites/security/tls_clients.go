package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

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
