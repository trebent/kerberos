package forwarder

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/zerologr"
)

// newTransport builds an [http.Transport] that enforces the TLS settings declared for a
// backend. It is called once per backend at component construction so that the underlying
// connection pool and TLS session cache are shared across all requests to that backend.
//
// When tlsCfg is nil the returned transport uses Go's defaults (plain HTTP).
// When fetcher is non-nil it is used to obtain the client certificate instead of loading
// it from the file paths in tlsCfg.
func newTransport(name string, tlsCfg *config.BackendTLS) (*http.Transport, error) {
	zerologr.Info("Loading backend transport", "backend", name)
	if tlsCfg == nil {
		return &http.Transport{}, nil
	}

	tc := &tls.Config{
		//nolint:gosec // InsecureSkipVerify is an explicit per-backend opt-in for non-production environments.
		InsecureSkipVerify: tlsCfg.InsecureSkipVerify,
	}

	if tlsCfg.RootCAFile != "" {
		zerologr.Info(
			"Loading backend root CA bundle",
			"backend", name,
			"path", tlsCfg.RootCAFile,
		)
		pem, err := os.ReadFile(tlsCfg.RootCAFile)
		if err != nil {
			return nil, fmt.Errorf("reading root CA bundle %q: %w", tlsCfg.RootCAFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no valid PEM certificates found in %q", tlsCfg.RootCAFile)
		}
		tc.RootCAs = pool
	}

	if tlsCfg.ClientCertFile != "" || tlsCfg.ClientKeyFile != "" {
		zerologr.Info(
			"Loading mTLS client key pair",
			"backend", name,
			"certFile", tlsCfg.ClientCertFile,
			"keyFile", tlsCfg.ClientKeyFile,
		)
		cert, err := tls.LoadX509KeyPair(tlsCfg.ClientCertFile, tlsCfg.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("loading mTLS client key pair: %w", err)
		}
		tc.Certificates = []tls.Certificate{cert}
	}

	return &http.Transport{TLSClientConfig: tc}, nil
}
