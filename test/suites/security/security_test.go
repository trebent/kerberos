package security

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// certDir is relative to the test working directory (test/suites/security/).
const certDir = "../../certs"

const (
	kerberosPort = 30000
	adminPort    = 30001
	echoPort     = 15000
)

// caPool loads the test CA certificate into a new cert pool.
func caPool(t *testing.T) *x509.CertPool {
	t.Helper()
	pem, err := os.ReadFile(certDir + "/ca.crt")
	if err != nil {
		t.Fatalf("Read CA cert: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		t.Fatal("No certificates found in ca.crt")
	}
	return pool
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

// mtlsClient returns an http.Client that both verifies the server cert against
// the test CA and presents the client cert/key pair for mTLS.
func mtlsClient(t *testing.T) *http.Client {
	t.Helper()
	cert, err := tls.LoadX509KeyPair(certDir+"/client.crt", certDir+"/client.key")
	if err != nil {
		t.Fatalf("Load client cert/key: %v", err)
	}
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caPool(t),
				Certificates: []tls.Certificate{cert},
			},
		},
	}
}

// plainClient returns an http.Client that uses plain HTTP (no TLS).
func plainClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

// ---- Admin API ----

// TestAdminAPITLS verifies that the admin API is reachable over TLS.
// A 401 response confirms the TLS handshake completed and the server is up.
func TestAdminAPITLS(t *testing.T) {
	resp, err := tlsClient(t).Get(fmt.Sprintf("https://localhost:%d/api/admin/flow", adminPort))
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// TestAdminAPIPlainHTTP verifies that the admin API rejects plain HTTP connections.
func TestAdminAPIPlainHTTP(t *testing.T) {
	resp, err := plainClient().Get(fmt.Sprintf("http://localhost:%d/api/admin/flow", adminPort))
	if err != nil {
		t.Fatalf("Unexpected error when sending plain HTTP request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request when sending plain HTTP to a TLS-only port, got %d", resp.StatusCode)
	}
}

// ---- Gateway API ----

// TestGWAPITLS verifies that the gateway API is reachable over TLS.
func TestGWAPITLS(t *testing.T) {
	resp, err := tlsClient(t).Get(fmt.Sprintf("https://localhost:%d/gw/backend/echo/hi", kerberosPort))
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

// TestGWAPIPlainHTTP verifies that the gateway API rejects plain HTTP connections.
func TestGWAPIPlainHTTP(t *testing.T) {
	resp, err := plainClient().Get(fmt.Sprintf("http://localhost:%d/gw/backend/echo/hi", kerberosPort))
	if err != nil {
		t.Fatalf("Unexpected error when sending plain HTTP request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request when sending plain HTTP to a TLS-only port, got %d", resp.StatusCode)
	}
}

// ---- Echo (mTLS) ----

// EchoResponse is the JSON body echo sends back.
type EchoResponse struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body,omitempty"`
}

// TestEchoMTLS verifies that echo accepts a connection when the client presents
// a valid certificate, and that echo itself presents a valid server certificate.
func TestEchoMTLS(t *testing.T) {
	resp, err := mtlsClient(t).Get(fmt.Sprintf("https://localhost:%d/hi", echoPort))
	if err != nil {
		t.Fatalf("mTLS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify the server actually presented a certificate signed by our CA.
	if resp.TLS == nil {
		t.Fatal("expected a TLS connection state, got nil")
	}
	if len(resp.TLS.PeerCertificates) == 0 {
		t.Fatal("expected echo to present at least one certificate")
	}
	serverCert := resp.TLS.PeerCertificates[0]
	if serverCert.Subject.CommonName != "echo" {
		t.Errorf("expected server cert CN %q, got %q", "echo", serverCert.Subject.CommonName)
	}
	if err := serverCert.VerifyHostname("localhost"); err != nil {
		t.Errorf("server cert does not cover 'localhost': %v", err)
	}
}

// TestEchoMTLSNoClientCert verifies that echo rejects TLS connections that do
// not supply a client certificate (mTLS is mandatory).
func TestEchoMTLSNoClientCert(t *testing.T) {
	_, err := tlsClient(t).Get(fmt.Sprintf("https://localhost:%d/hi", echoPort))
	if err == nil {
		t.Fatal("expected a TLS error when connecting without a client certificate, got nil")
	}
}

// TestEchoPlainHTTP verifies that echo rejects plain HTTP connections.
func TestEchoPlainHTTP(t *testing.T) {
	_, err := plainClient().Get(fmt.Sprintf("http://localhost:%d/hi", echoPort))
	if err == nil {
		t.Fatal("expected a transport error when sending plain HTTP to a TLS-only port, got nil")
	}
}
