// Package router implements routing support for the KRB service. It defines how to register
// and fetch available backends.
package router

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/trebent/kerberos/internal/config"
)

type (
	// Router is used to route incoming requests to matching backends. Failure to route is terminal
	// and will yield a 404.
	Router interface {
		// GetBackend returns the backend for the given request. The backend is determined by
		// matching the request to the registered backends. If no backend is found, an error is
		// returned.
		GetBackend(http.Request) (Backend, error)
	}
	// Opts are the options used to configure the router.
	Opts struct {
		Cfg config.Map
	}
	router struct {
		cfg *routerConfig
	}
	routingCtxKey string
)

var (
	routePattern = regexp.MustCompile(`^/gw/backend/([-_a-z0-9]+)?/.+$`)

	ErrFailedPatternMatch = errors.New("backend pattern match failed")
	ErrNoBackendFound     = errors.New("no backend found")
)

const expectedPatternMatches = 2

// New returns a router based on the provided options.
func New(opts *Opts) Router {
	return &router{config.AccessAs[*routerConfig](opts.Cfg, configName)}
}

func (r *router) GetBackend(req http.Request) (Backend, error) {
	reqPath := routePattern.FindStringSubmatch(req.URL.Path)

	if len(reqPath) < expectedPatternMatches {
		return nil, fmt.Errorf("%w: %s", ErrFailedPatternMatch, req.URL.Path)
	}

	for _, backend := range r.cfg.Backends {
		if backend.Name() == reqPath[1] {
			return backend, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrNoBackendFound, req.URL.Path)
}
