package adminext

import (
	"net/http"

	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
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

	// APIProvider is implemented by any extension that wants to expose additional admin API endpoints.
	APIProvider interface {
		// RegisterRoutes allows the extension to register its own HTTP handlers on the provided ServeMux.
		// The extension should use a unique path prefix to avoid conflicts with other extensions.
		RegisterRoutes(mux *http.ServeMux)
	}
)
