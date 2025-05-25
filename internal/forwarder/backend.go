// nolint // testing purposes
package forwarder

import "github.com/trebent/kerberos/internal/router"

type TestBackend struct{}

var _ router.Backend = &TestBackend{}

// Host implements router.Backend.
func (t *TestBackend) Host() string {
	return "test-host"
}

// Name implements router.Backend.
func (t *TestBackend) Name() string {
	return "test-backend"
}

// Port implements router.Backend.
func (t *TestBackend) Port() int {
	return 8080
}
