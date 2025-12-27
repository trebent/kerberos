package router

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-logr/logr"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/response"
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
)

var (
	routePattern                             = regexp.MustCompile(`^/gw/backend/([-_a-z0-9]+)?/.+$`)
	_            composertypes.FlowComponent = (*router)(nil)

	ErrFailedPatternMatch = errors.New("backend pattern match failed")
	ErrNoBackendFound     = errors.New("no backend found")
)

const expectedPatternMatches = 2

func NewComponent(_ Opts) composertypes.FlowComponent {
	return &router{}
}

// Next implements [types.FlowComponent].
func (r *router) Next(_ composertypes.FlowComponent) {
	panic("unimplemented")
}

// ServeHTTP implements [types.FlowComponent].
func (r *router) ServeHTTP(http.ResponseWriter, *http.Request) {
	panic("unimplemented")
}

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

func Middleware(next http.Handler, router Router) http.Handler {
	return http.HandlerFunc(func(wrapped http.ResponseWriter, r *http.Request) {
		logger, _ := logr.FromContext(r.Context())
		rLogger := logger.WithName("router")
		rLogger.Info("Routing request")

		backend, err := router.GetBackend(*r)
		if err != nil {
			rLogger.Error(err, "Failed to route request")
			response.JSONError(wrapped, ErrNoBackendFound, http.StatusNotFound)
			return
		}

		// Set backend in context logger to forward. Don't append to the name.
		ctx := logr.NewContext(r.Context(), logger.WithValues("backend", backend.Name()))
		ctx = NewBackendContext(ctx, backend)

		// Update the wrapper request context to be able to extract in higher level middleware.
		wrapper, _ := wrapped.(*response.Wrapper)
		wrapper.SetRequestContext(ctx)

		// Serve the request with the updated context.
		next.ServeHTTP(wrapped, r.WithContext(ctx))
	})
}
