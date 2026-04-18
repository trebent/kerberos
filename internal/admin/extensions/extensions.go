package adminext

import (
	"net/http"

	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
)

type (
	// FlowFetcher implementors provide a way for the admin API to fetch the FlowComponent chain.
	FlowFetcher interface {
		// GetFlow returns metadata about the current FlowComponent chain. This is used for flow inspection in the admin API.
		GetFlow() []adminapi.FlowMeta
	}
	// OASBackend implementors provide a way for the admin API to fetch an OAS per backend name.
	OASBackend interface {
		// GetOAS returns the OpenAPI Specification for the specified backend.
		GetOAS(backendName string) ([]byte, error)
	}
	// DummyOASBackend is a no-op OAS backend that always returns not found. This is used by default
	// when admin is instantiated without an OAS backend, to avoid nil checks.
	DummyOASBackend struct{}

	// APIProvider is implemented by any extension that wants to expose additional admin API endpoints.
	APIProvider interface {
		// RegisterRoutes allows the extension to register its own HTTP handlers on the provided ServeMux.
		// The extension should use a unique path prefix to avoid conflicts with other extensions.
		RegisterRoutes(
			mux *http.ServeMux,
			middleware ...strictnethttp.StrictHTTPMiddlewareFunc,
		) error
	}
)

var _ OASBackend = (*DummyOASBackend)(nil)

func (d *DummyOASBackend) GetOAS(_ string) ([]byte, error) {
	return nil, apierror.ErrNotFound
}
