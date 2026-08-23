package lib

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

// AdminResponsesTLSClient returns an adminapi.ClientWithResponses that verifies the server cert against
// the test CA but sends no client certificate. certDir is the path to the directory containing ca.crt.
func AdminResponsesTLSClient(t *testing.T, certDir string) *adminapi.ClientWithResponses {
	t.Helper()
	client, err := adminapi.NewClientWithResponses(
		fmt.Sprintf("https://%s:%d", GetHost(), GetAdminPort()),
		adminapi.WithHTTPClient(TLSClient(t, certDir)),
	)
	CheckErr(err, t)
	return client
}

// BasicAuthResponsesTLSClient returns an authbasicapi.ClientWithResponses that verifies the server cert against
// the test CA but sends no client certificate. certDir is the path to the directory containing ca.crt.
func BasicAuthResponsesTLSClient(t *testing.T, certDir string) *authbasicapi.ClientWithResponses {
	t.Helper()
	client, err := authbasicapi.NewClientWithResponses(
		fmt.Sprintf("https://%s:%d", GetHost(), GetAdminPort()),
		authbasicapi.WithHTTPClient(TLSClient(t, certDir)),
	)
	CheckErr(err, t)
	return client
}

// TLSClient returns an http.Client that verifies the server cert against the
// test CA but sends no client certificate. certDir is the path to the directory containing ca.crt.
func TLSClient(t *testing.T, certDir string) *http.Client {
	t.Helper()
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: CAPool(t, certDir),
			},
		},
	}
}

// PlainClient returns an http.Client that uses plain HTTP (no TLS).
func PlainClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

// CAPool loads the test CA certificate into a new cert pool.
// certDir is the path to the directory containing ca.crt.
func CAPool(t *testing.T, certDir string) *x509.CertPool {
	t.Helper()
	pool, err := GetCAPool(certDir)
	if err != nil {
		t.Fatalf("Failed to load CA pool: %v", err)
	}
	return pool
}

// GetCAPool loads the CA certificate from certDir/ca.crt and returns a cert pool.
func GetCAPool(certDir string) (*x509.CertPool, error) {
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
