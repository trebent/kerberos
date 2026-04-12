package forwarder_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/forwarder"
	"github.com/trebent/kerberos/internal/config"
)

func TestForwarder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	split := strings.Split(server.URL, ":")
	host := split[1][2:]
	port := split[2]

	t.Log("Host:", host)
	t.Log("Port:", port)

	intPort, _ := strconv.Atoi(port)

	// Not strictly necessary to pass router config, since it's only used to init TLS conf.
	// But it makes the test more realistic and ensures the forwarder can be initialized with a typical config.
	fwd, err := forwarder.NewComponent(&forwarder.Opts{
		Backends: []*config.RouterBackend{
			{
				Name: "test-backend",
				Host: host,
				Port: intPort,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create forwarder component: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "/test"},
	}
	ctx := context.WithValue(request.Context(), composer.TargetContextKey, &config.RouterBackend{
		Name: "test-backend",
		Host: host,
		Port: intPort,
	})

	fwd.ServeHTTP(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestForwarderTLS(t *testing.T) {
	var servedOverTLS bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		servedOverTLS = r.TLS != nil
		t.Log("Received request over TLS:", servedOverTLS)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// PEM-encode the server's self-signed certificate so the forwarder can trust it.
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	caFile, err := os.CreateTemp(t.TempDir(), "ca-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp CA file: %v", err)
	}
	if _, err := caFile.Write(certPEM); err != nil {
		t.Fatalf("Failed to write CA PEM: %v", err)
	}
	caFile.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Failed to parse server URL: %v", err)
	}
	host := serverURL.Hostname()
	port, _ := strconv.Atoi(serverURL.Port())

	backend := &config.RouterBackend{
		Name: "tls-backend",
		Host: host,
		Port: port,
		TLS:  &config.BackendTLS{RootCAFile: caFile.Name()},
	}

	fwd, err := forwarder.NewComponent(&forwarder.Opts{
		Backends: []*config.RouterBackend{backend},
	})
	if err != nil {
		t.Fatalf("Failed to create forwarder component: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "/test"},
	}
	ctx := context.WithValue(request.Context(), composer.TargetContextKey, backend)

	fwd.ServeHTTP(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
	if !servedOverTLS {
		t.Error("Expected request to be served over TLS, but r.TLS was nil on the backend")
	}
}

func TestForwarderTLSUntrustedCA(t *testing.T) {
	// The actual backend the forwarder will contact.
	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	// httptest.NewTLSServer always uses the same hardcoded certificate, so a second test
	// server would share the same cert. Instead, generate a fresh self-signed certificate
	// that is entirely unrelated to the backend's certificate.
	unrelatedCAPEM := generateUnrelatedCACert(t)
	caFile, err := os.CreateTemp(t.TempDir(), "wrong-ca-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp CA file: %v", err)
	}
	if _, err := caFile.Write(unrelatedCAPEM); err != nil {
		t.Fatalf("Failed to write CA PEM: %v", err)
	}
	caFile.Close()

	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatalf("Failed to parse backend URL: %v", err)
	}
	host := backendURL.Hostname()
	port, _ := strconv.Atoi(backendURL.Port())

	backendCfg := &config.RouterBackend{
		Name: "untrusted-tls-backend",
		Host: host,
		Port: port,
		TLS:  &config.BackendTLS{RootCAFile: caFile.Name()},
	}

	fwd, err := forwarder.NewComponent(&forwarder.Opts{
		Backends: []*config.RouterBackend{backendCfg},
	})
	if err != nil {
		t.Fatalf("Failed to create forwarder component: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "/test"},
	}
	ctx := context.WithValue(request.Context(), composer.TargetContextKey, backendCfg)

	fwd.ServeHTTP(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d due to TLS verification failure, got %d", http.StatusInternalServerError, recorder.Code)
	}
}

func TestForwarderMTLS(t *testing.T) {
	ca := newTestCA(t)

	// Write the CA cert so the forwarder can verify the server's certificate.
	caFile, err := os.CreateTemp(t.TempDir(), "ca-*.pem")
	if err != nil {
		t.Fatalf("Failed to create CA file: %v", err)
	}
	if _, err := caFile.Write(ca.certPEM); err != nil {
		t.Fatalf("Failed to write CA PEM: %v", err)
	}
	caFile.Close()

	// Server cert must include 127.0.0.1 as an IP SAN since httptest binds to that address.
	serverCert := ca.signCert(t, 2, []net.IP{net.ParseIP("127.0.0.1")})
	clientCert := ca.signCert(t, 3, nil)

	// Server trusts our CA for client certificate verification.
	clientCAPool := x509.NewCertPool()
	clientCAPool.AddCert(ca.cert)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert.tlsCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAPool,
	}
	server.StartTLS()
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Failed to parse server URL: %v", err)
	}
	host := serverURL.Hostname()
	port, _ := strconv.Atoi(serverURL.Port())

	backend := &config.RouterBackend{
		Name: "mtls-backend",
		Host: host,
		Port: port,
		TLS: &config.BackendTLS{
			RootCAFile:     caFile.Name(),
			ClientCertFile: clientCert.certFile,
			ClientKeyFile:  clientCert.keyFile,
		},
	}

	fwd, err := forwarder.NewComponent(&forwarder.Opts{
		Backends: []*config.RouterBackend{backend},
	})
	if err != nil {
		t.Fatalf("Failed to create forwarder component: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "/test"},
	}
	ctx := context.WithValue(request.Context(), composer.TargetContextKey, backend)

	fwd.ServeHTTP(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestForwarderMTLSUntrustedClientCert(t *testing.T) {
	serverCA := newTestCA(t)
	caFile, err := os.CreateTemp(t.TempDir(), "ca-*.pem")
	if err != nil {
		t.Fatalf("Failed to create CA file: %v", err)
	}
	if _, err := caFile.Write(serverCA.certPEM); err != nil {
		t.Fatalf("Failed to write CA PEM: %v", err)
	}
	caFile.Close()

	// A separate CA signs the client cert — the server has no knowledge of it.
	attackerCA := newTestCA(t)

	serverCert := serverCA.signCert(t, 2, []net.IP{net.ParseIP("127.0.0.1")})
	untrustedClientCert := attackerCA.signCert(t, 2, nil)

	clientCAPool := x509.NewCertPool()
	clientCAPool.AddCert(serverCA.cert)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert.tlsCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAPool,
	}
	server.StartTLS()
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Failed to parse server URL: %v", err)
	}
	host := serverURL.Hostname()
	port, _ := strconv.Atoi(serverURL.Port())

	backend := &config.RouterBackend{
		Name: "mtls-untrusted-backend",
		Host: host,
		Port: port,
		TLS: &config.BackendTLS{
			RootCAFile:     caFile.Name(),
			ClientCertFile: untrustedClientCert.certFile,
			ClientKeyFile:  untrustedClientCert.keyFile,
		},
	}

	fwd, err := forwarder.NewComponent(&forwarder.Opts{
		Backends: []*config.RouterBackend{backend},
	})
	if err != nil {
		t.Fatalf("Failed to create forwarder component: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "/test"},
	}
	ctx := context.WithValue(request.Context(), composer.TargetContextKey, backend)

	fwd.ServeHTTP(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d due to mTLS client cert rejection, got %d", http.StatusInternalServerError, recorder.Code)
	}
}

// generateUnrelatedCACert creates a fresh self-signed CA certificate unrelated to any
// httptest server certificate and returns its PEM encoding.
func generateUnrelatedCACert(t *testing.T) []byte {
	t.Helper()
	return newTestCA(t).certPEM
}

// testCA holds a generated certificate authority used to sign test certificates.
type testCA struct {
	cert    *x509.Certificate
	certPEM []byte
	key     *ecdsa.PrivateKey
}

// newTestCA creates a fresh self-signed CA for use in TLS tests.
func newTestCA(t *testing.T) testCA {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate CA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("Failed to create CA certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		t.Fatalf("Failed to parse CA certificate: %v", err)
	}
	return testCA{
		cert:    cert,
		certPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes}),
		key:     priv,
	}
}

// signedCert holds a certificate signed by a [testCA] and exposes both the on-disk
// paths (for the forwarder's file-based config) and the loaded [tls.Certificate]
// (for configuring test servers directly).
type signedCert struct {
	certFile string
	keyFile  string
	tlsCert  tls.Certificate
}

// signCert issues a certificate signed by ca. Pass IP SANs for server certificates
// (e.g. net.ParseIP("127.0.0.1")); client certificates do not need SANs.
func (ca testCA) signCert(t *testing.T, serial int64, ipSANs []net.IP) signedCert {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate cert key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: "test-cert"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  ipSANs,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &priv.PublicKey, ca.key)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("Failed to load X509 key pair: %v", err)
	}

	dir := t.TempDir()
	cf, err := os.CreateTemp(dir, "cert-*.pem")
	if err != nil {
		t.Fatalf("Failed to create cert file: %v", err)
	}
	cf.Write(certPEM)
	cf.Close()

	kf, err := os.CreateTemp(dir, "key-*.pem")
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	kf.Write(keyPEM)
	kf.Close()

	return signedCert{certFile: cf.Name(), keyFile: kf.Name(), tlsCert: tlsCert}
}
