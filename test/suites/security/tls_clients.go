package security

import (
	"crypto/x509"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
	lib "github.com/trebent/kerberos/test/lib"
)

func adminResponsesTLSClient(t *testing.T) *adminapi.ClientWithResponses {
	t.Helper()
	return lib.AdminResponsesTLSClient(t, certDir)
}

func basicAuthResponsesTLSClient(t *testing.T) *authbasicapi.ClientWithResponses {
	t.Helper()
	return lib.BasicAuthResponsesTLSClient(t, certDir)
}

func tlsClient(t *testing.T) *http.Client {
	t.Helper()
	return lib.TLSClient(t, certDir)
}

func plainClient() *http.Client {
	return lib.PlainClient()
}

func getCAPool() (*x509.CertPool, error) {
	return lib.GetCAPool(certDir)
}
