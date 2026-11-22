package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/go-logr/logr"
	adminext "github.com/trebent/kerberos/internal/admin/extensions"
	"github.com/trebent/kerberos/internal/composer"
	composerdebug "github.com/trebent/kerberos/internal/composer/debug"
	"github.com/trebent/kerberos/internal/config"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/kerberos/internal/security"
	"github.com/trebent/zerologr"
)

type (
	// Opts are the options used to configure the router.
	Opts struct {
		Cfg *config.Router
	}
	router struct {
		cfg  *config.Router
		next composer.FlowComponent
	}
)

var (
	_ composer.FlowComponent = (*router)(nil)
	_ adminext.BackendLookup = (*router)(nil)

	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	apiErrNoBackendFound = apierror.New(http.StatusNotFound, "no backend found")
)

const (
	expectedPatternMatches = 2
	prefix                 = "/gw/backend/"
)

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

// BackendExists implements [adminext.BackendLookup]
func (r *router) BackendExists(name string) bool {
	return slices.ContainsFunc(r.cfg.Backends, func(backend *config.RouterBackend) bool {
		return name == backend.Name
	})
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
	debugStart := time.Now()
	debuggedCall := composer.DebugFromContext(req.Context())

	logger, _ := logr.FromContext(req.Context())

	if l := logger.V(20); l.Enabled() {
		l.Info("Routing request", "path", req.URL.Path)
	}

	backend, err := r.GetBackend(req)
	if errors.Is(err, apiErrNoBackendFound) {
		logger.Error(err, "Failed to route request")
		apierror.ErrorHandler(wrapped, req, apiErrNoBackendFound)
		debuggedCall.AddTransition(
			"router",
			composerdebug.DirectionInbound,
			debugStart,
			time.Now(),
			composerdebug.ResultFailure,
			apiErrNoBackendFound.Error(),
		)
		return
	}

	// Vary on Origin for requests without one too, see security.AddVaryOrigin.
	cors := corsConfigured(backend)
	if cors {
		security.AddVaryOrigin(wrapped)
	}

	if security.HasOrigin(req) && cors {
		origin := req.Header.Get("Origin")
		if err := ValidateOrigin(
			origin,
			backend.Origins,
		); err != nil {
			logger.Error(err, "Origin denied", "origin", origin)
			apierror.ErrorHandler(wrapped, req, err)
			debuggedCall.AddTransition(
				"router",
				composerdebug.DirectionInbound,
				debugStart,
				time.Now(),
				composerdebug.ResultFailure,
				err.Error(),
			)
			return
		}

		security.SetCORSHeaders(wrapped, req.Header.Get("Origin"))

		if req.Method == http.MethodOptions {
			wrapped.WriteHeader(http.StatusNoContent)
			debuggedCall.AddTransition(
				"router",
				composerdebug.DirectionInbound,
				debugStart,
				time.Now(),
				composerdebug.ResultSuccess,
				"",
			)
			return
		}
	}

	// Set backend in context logger to forward. Don't append to the name.
	ctx := logr.NewContext(req.Context(), logger.WithValues("backend", backend.Name))
	ctx = context.WithValue(ctx, composer.TargetContextKey, backend)

	// Update the wrapper request context to be able to extract in higher level middleware.
	wrapper, _ := wrapped.(*response.Wrapper)
	wrapper.SetRequestContext(ctx)

	// Strip the /gw/backend/{backend-name} prefix from the request URL path.
	req.URL.Path = stripKrbPrefix(req.URL.Path, backend.Name)

	debuggedCall.AddTransition(
		"router",
		composerdebug.DirectionInbound,
		debugStart,
		time.Now(),
		composerdebug.ResultSuccess,
		"",
	)

	// Serve the request with the updated context.
	r.next.ServeHTTP(wrapped, req.WithContext(ctx))
}

func (r *router) GetBackend(req *http.Request) (*config.RouterBackend, error) {
	backendName := req.Context().Value(composer.BackendContextKey)

	for _, backend := range r.cfg.Backends {
		if backend.Name == backendName {
			return backend, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", apiErrNoBackendFound, req.URL.Path)
}

// stripKrbPrefix strips the /gw/backend/{backend-name} prefix from the request URL path.
func stripKrbPrefix(path, backend string) string {
	return path[len(prefix)+len(backend):]
}
