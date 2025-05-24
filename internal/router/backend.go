package router

import (
	"context"
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

const backendContextKey routingCtxKey = "backend"

func NewBackendContext(ctx context.Context, value any) context.Context {
	return context.WithValue(ctx, backendContextKey, value)
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
