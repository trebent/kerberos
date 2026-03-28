package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-logr/logr"
	adminapi "github.com/trebent/kerberos/internal/api/admin"
	apierror "github.com/trebent/kerberos/internal/api/error"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/zerologr"
)

type (
	// Opts are the options used to configure the router.
	Opts struct {
		Cfg *config.RouterConfig
	}
	router struct {
		cfg  *config.RouterConfig
		next composer.FlowComponent
	}
)

var (
	routePattern                        = regexp.MustCompile(`^/gw/backend/([-_a-z0-9]+)?/.*$`)
	_            composer.FlowComponent = (*router)(nil)

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

func NewBackendContext(ctx context.Context, backend *config.RouterBackend) context.Context {
	ctx = context.WithValue(ctx, composer.TargetContextKey, backend)
	return context.WithValue(ctx, composer.BackendContextKey, backend.Name)
}

func NewComponent(opts *Opts) composer.FlowComponent {
	for _, backend := range opts.Cfg.Backends {
		zerologr.Info(
			"Configured backend",
			"backend", backend.Name,
			"host", backend.Host,
			"port", backend.Port,
		)
	}
	return &router{
		cfg: opts.Cfg,
	}
}

// Next implements [composer.FlowComponent].
func (r *router) Next(next composer.FlowComponent) {
	r.next = next
}

// GetMeta implements [composer.FlowComponent].
func (r *router) GetMeta() []adminapi.FlowMeta {
	fmd := adminapi.FlowMeta_Data{}
	if err := fmd.FromFlowMetaDataRouter(adminapi.FlowMetaDataRouter{
		Backends: func() *[]adminapi.FlowMetaDataRouterBackend {
			var backends []adminapi.FlowMetaDataRouterBackend
			for _, backend := range r.cfg.Backends {
				backends = append(backends, adminapi.FlowMetaDataRouterBackend{
					Name: backend.Name,
					Host: backend.Host,
					Port: backend.Port,
				})
			}
			return &backends
		}(),
	}); err != nil {
		panic(err)
	}

	return append([]adminapi.FlowMeta{
		{
			Name: "router",
			Data: fmd,
		},
	}, r.next.GetMeta()...)
}

// ServeHTTP implements [composer.FlowComponent].
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
	ctx := logr.NewContext(req.Context(), logger.WithValues("backend", backend.Name))
	ctx = NewBackendContext(ctx, backend)

	// Update the wrapper request context to be able to extract in higher level middleware.
	wrapper, _ := wrapped.(*response.Wrapper)
	wrapper.SetRequestContext(ctx)

	// Strip the /gw/backend/{backend-name} prefix from the request URL path.
	req.URL.Path = stripKrbPrefix(req.URL.Path, backend.Name)

	// Serve the request with the updated context.
	r.next.ServeHTTP(wrapped, req.WithContext(ctx))
}

func (r *router) GetBackend(req http.Request) (*config.RouterBackend, error) {
	reqPath := routePattern.FindStringSubmatch(req.URL.Path)

	if len(reqPath) < expectedPatternMatches {
		return nil, fmt.Errorf("%w: %s", errFailedPatternMatch, req.URL.Path)
	}

	for _, backend := range r.cfg.Backends {
		if backend.Name == reqPath[1] {
			return backend, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", errNoBackendFound, req.URL.Path)
}

// stripKrbPrefix strips the /gw/backend/{backend-name} prefix from the request URL path.
func stripKrbPrefix(path, backend string) string {
	return path[len(prefix)+len(backend):]
}
