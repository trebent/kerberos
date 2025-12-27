package router

import (
	"context"

	composertypes "github.com/trebent/kerberos/internal/composer/types"
)

type (
	// Backend is a target backend service that KRB forwards requests to.
	Backend interface {
		Name() string
		Host() string
		Port() int
	}
	backend struct {
		BackendName string `json:"name"`
		BackendHost string `json:"host"`
		BackendPort int    `json:"port"`
	}
)

var _ Backend = &backend{}

func NewBackendContext(ctx context.Context, backend Backend) context.Context {
	ctx = context.WithValue(ctx, composertypes.TargetContextKey, backend)
	return context.WithValue(ctx, composertypes.BackendContextKey, backend.Name())
}

func (b *backend) Name() string {
	return b.BackendName
}

func (b *backend) Host() string {
	return b.BackendHost
}

func (b *backend) Port() int {
	return b.BackendPort
}
