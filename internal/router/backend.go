package router

import (
	"context"

	"github.com/trebent/kerberos/internal/otel"
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

const backendContextKey routingCtxKey = "routing.backend"

func NewBackendContext(ctx context.Context, backend Backend) context.Context {
	ctx = context.WithValue(ctx, backendContextKey, backend)
	return context.WithValue(ctx, otel.KrbMetaBackend, backend.Name())
}

func BackendFromContext(ctx context.Context) Backend {
	backend, _ := ctx.Value(backendContextKey).(Backend)
	return backend
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
