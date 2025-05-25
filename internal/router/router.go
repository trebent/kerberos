// Package router implements routing support for the KRB service. It defines how to register
// and fetch available backends.
package router

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
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
	// RouterOpts are the options used to configure the router.
	RouterOpts struct {
		// Specify the loader to use for loading backends. The loader is responsible for
		// loading the backends from a source. The source can be a file, a database, etc.
		Loader BackendLoader
	}
	router struct {
		backends []Backend
	}
	routingCtxKey string
)

var (
	routePattern = regexp.MustCompile(`^/gw/backend/([-_a-z0-9]+)?/.+$`)

	ErrFailedPatternMatch = errors.New("backend pattern match failed")
	ErrNoBackendFound     = errors.New("no backend found")
)

// Load a router based on the provided option's loader.
func Load(opts *RouterOpts) (Router, error) {
	backends, err := opts.Loader.Load()
	if err != nil {
		return nil, err
	}

	r := &router{backends: backends}
	return r, nil
}

func (r *router) GetBackend(req http.Request) (Backend, error) {
	reqPath := routePattern.FindStringSubmatch(req.URL.Path)

	if len(reqPath) < 2 {
		return nil, fmt.Errorf("%w: %s", ErrFailedPatternMatch, req.URL.Path)
	}

	for _, backend := range r.backends {
		if backend.Name() == reqPath[1] {
			return backend, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrNoBackendFound, req.URL.Path)
}
