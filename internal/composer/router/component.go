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
	// Opts are the options used to configure the router.
	Opts struct {
		Cfg config.Map
	}
	router struct {
		cfg  *routerConfig
		next composertypes.FlowComponent
	}
)

var (
	routePattern                             = regexp.MustCompile(`^/gw/backend/([-_a-z0-9]+)?/.*$`)
	_            composertypes.FlowComponent = (*router)(nil)

	ErrFailedPatternMatch = errors.New("backend pattern match failed")
	ErrNoBackendFound     = errors.New("no backend found")
)

const expectedPatternMatches = 2

func NewComponent(opts *Opts) composertypes.FlowComponent {
	return &router{cfg: config.AccessAs[*routerConfig](opts.Cfg, configName)}
}

// Next implements [types.FlowComponent].
func (r *router) Next(next composertypes.FlowComponent) {
	r.next = next
}

// ServeHTTP implements [types.FlowComponent].
func (r *router) ServeHTTP(wrapped http.ResponseWriter, req *http.Request) {
	logger, _ := logr.FromContext(req.Context())
	rLogger := logger.WithName("router")
	rLogger.Info("Routing request")

	backend, err := r.GetBackend(*req)
	if errors.Is(err, ErrNoBackendFound) {
		rLogger.Error(err, "Failed to route request")
		response.JSONError(wrapped, ErrNoBackendFound, http.StatusNotFound)
		return
	} else if errors.Is(err, ErrFailedPatternMatch) {
		rLogger.Error(err, "Failed to route request")
		response.JSONError(
			wrapped,
			fmt.Errorf("%w: backend path must begin with /gw/backend/backend-name", ErrFailedPatternMatch),
			http.StatusBadRequest,
		)
		return
	}

	// Set backend in context logger to forward. Don't append to the name.
	ctx := logr.NewContext(req.Context(), logger.WithValues("backend", backend.Name()))
	ctx = NewBackendContext(ctx, backend)

	// Update the wrapper request context to be able to extract in higher level middleware.
	wrapper, _ := wrapped.(*response.Wrapper)
	wrapper.SetRequestContext(ctx)

	// Serve the request with the updated context.
	r.next.ServeHTTP(wrapped, req.WithContext(ctx))
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
