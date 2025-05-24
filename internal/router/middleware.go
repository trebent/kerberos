package router

import (
	"net/http"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/response"
)

func Middleware(next http.Handler, router Router) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger, _ := logr.FromContext(r.Context())
		rLogger := logger.WithName("router")
		rLogger.Info("Routing request")

		backend, err := router.GetBackend(*r)
		if err != nil {
			rLogger.Error(err, "Failed to route request")
			jsonError, _ := response.JSONError("backend not found")
			http.Error(w, string(jsonError), http.StatusNotFound)
			return
		}

		// Set backend in context logger to forward. Don't append to the name.
		ctx := logr.NewContext(r.Context(), logger.WithValues("backend", backend.Name()))
		ctx = NewBackendContext(ctx, backend)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
