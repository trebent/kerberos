package router

import (
	"context"

	composerctx "github.com/trebent/kerberos/internal/composer/context"
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
	ctx = context.WithValue(ctx, composerctx.TargetContextKey, backend)
	return context.WithValue(ctx, composerctx.BackendContextKey, backend.Name())
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
