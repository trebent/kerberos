package security

import (
	"net/http"

	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/zerologr"
)

// SelectCORSMiddleware selects the appropriate CORS middleware based on the provided configuration.
// If allowAll is true, it returns the CORSMiddleware that allows all origins.
// If allowedOrigins is non-empty, it returns the WhitelistCORSMiddleware that allows only the specified origins.
// If neither condition is met, it returns the next handler without any CORS middleware.
func SelectCORSMiddleware(allowedOrigins []string, allowAll bool, next http.Handler) http.Handler {
	if allowAll {
		return CORSMiddleware(next)
	}

	if len(allowedOrigins) > 0 {
		return WhitelistCORSMiddleware(allowedOrigins, next)
	}

	return next
}

// CORSMiddleware is a middleware that adds CORS headers to the response.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zerologr.V(20).Info("CORS middleware: attaching CORS headers")
		if r.Header.Get("Origin") == "" {
			// No Origin header present, so this is not a browser request.
			zerologr.V(20).Info("CORS middleware: not a browser request, skipping CORS headers")
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		// Read more here for whitelisted headers: https://developer.mozilla.org/en-US/docs/Glossary/CORS-safelisted_response_header
		w.Header().Set("Access-Control-Allow-Headers", CSRFTokenHeader)
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WhitelistCORSMiddleware is a middleware that adds CORS headers to the response for requests from allowed origins.
// Non-whitelisted origins will receive a 403 Forbidden response.
func WhitelistCORSMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zerologr.V(20).Info("Whitelist CORS middleware: checking allowed origins")
		if r.Header.Get("Origin") == "" {
			// No Origin header present, so this is not a browser request.
			zerologr.V(20).Info(
				"Whitelist CORS middleware: not a browser request, skipping CORS headers",
			)
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set(
					"Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS",
				)
				w.Header().Set("Access-Control-Allow-Headers", CSRFTokenHeader)
				w.Header().Set("Access-Control-Allow-Credentials", "true")

				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusOK)
					return
				}

				next.ServeHTTP(w, r)
				return
			}
		}

		// If the origin is not allowed, return a 403 Forbidden response.
		apierror.ErrorHandler(w, r, apierror.ErrForbidden)
	})
}
