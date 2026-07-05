package security

import (
	"net/http"

	"github.com/trebent/zerologr"
)

// CORSMiddleware is a middleware that adds CORS headers to the response.
// TODO: implement whitelist support for allowed origins.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zerologr.V(20).Info("CORS middleware: checking request for CORS headers")
		if r.Header.Get("Origin") == "" {
			// No Origin header present, so this is not a browser request.
			zerologr.V(20).Info("CORS middleware: not a browser request, skipping CORS headers")
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, "+CSRFTokenHeader)
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
