package router

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-logr/logr"
	apierror "github.com/trebent/kerberos/internal/api/error"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/zerologr"
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

	errFailedPatternMatch = errors.New("bad backend pattern")
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	apiErrFailedPatternMatch = apierror.New(http.StatusBadRequest, errFailedPatternMatch.Error())
	errNoBackendFound        = errors.New("no backend found")
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	apiErrNoBackendFound = apierror.New(http.StatusNotFound, errNoBackendFound.Error())
)

const (
	expectedPatternMatches = 2
	prefix                 = "/gw/backend/"
)

func NewComponent(opts *Opts) composertypes.FlowComponent {
	cfg := config.AccessAs[*routerConfig](opts.Cfg, configName)
	for _, backend := range cfg.Backends {
		zerologr.Info(
			"Configured backend",
			"backend", backend.Name(),
			"host", backend.Host(),
			"port", backend.Port(),
		)
	}
	return &router{cfg: cfg}
}

// Next implements [types.FlowComponent].
func (r *router) Next(next composertypes.FlowComponent) {
	r.next = next
}

// ServeHTTP implements [types.FlowComponent].
func (r *router) ServeHTTP(wrapped http.ResponseWriter, req *http.Request) {
	logger, _ := logr.FromContext(req.Context())
	rLogger := logger.WithName("router")
	rLogger.Info("Routing request", "path", req.URL.Path)

	backend, err := r.GetBackend(*req)
	if errors.Is(err, errNoBackendFound) {
		rLogger.Error(err, "Failed to route request")
		apierror.ErrorHandler(wrapped, req, apiErrNoBackendFound)
		return
	} else if errors.Is(err, errFailedPatternMatch) {
		rLogger.Error(err, "Failed to route request")
		apierror.ErrorHandler(wrapped, req, apiErrFailedPatternMatch)
		return
	}

	// Set backend in context logger to forward. Don't append to the name.
	ctx := logr.NewContext(req.Context(), logger.WithValues("backend", backend.Name()))
	ctx = NewBackendContext(ctx, backend)

	// Update the wrapper request context to be able to extract in higher level middleware.
	wrapper, _ := wrapped.(*response.Wrapper)
	wrapper.SetRequestContext(ctx)

	// Strip the /gw/backend/{backend-name} prefix from the request URL path.
	req.URL.Path = stripKrbPrefix(req.URL.Path, backend.Name())

	// Serve the request with the updated context.
	r.next.ServeHTTP(wrapped, req.WithContext(ctx))
}

func (r *router) GetBackend(req http.Request) (Backend, error) {
	reqPath := routePattern.FindStringSubmatch(req.URL.Path)

	if len(reqPath) < expectedPatternMatches {
		return nil, fmt.Errorf("%w: %s", errFailedPatternMatch, req.URL.Path)
	}

	for _, backend := range r.cfg.Backends {
		if backend.Name() == reqPath[1] {
			return backend, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", errNoBackendFound, req.URL.Path)
}

func stripKrbPrefix(path, backend string) string {
	return path[len(prefix)+len(backend):]
}
