package adminext

import adminapi "github.com/trebent/kerberos/internal/api/admin"

type (
	FlowFetcher interface {
		// GetFlow returns metadata about the current FlowComponent chain. This is used for flow inspection in the admin API.
		GetFlow() []adminapi.FlowMeta
	}
	OASBackend interface {
		// GetOAS returns the OpenAPI Specification for the specified backend.
		GetOAS(backendName string) ([]byte, error)
	}
)
