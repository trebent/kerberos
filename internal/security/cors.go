package security

import (
	"net/http"

	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/zerologr"
)

// SelectCORSMiddleware selects the appropriate CORS middleware based on the provided configuration.
// If denyAll is true, it returns the DenyAllOriginsMiddleware that denies all origins.
// If allowAll is true, it returns the CORSMiddleware that allows all origins.
// If allowedOrigins is non-empty, it returns the WhitelistCORSMiddleware that allows only the specified origins.
// If neither condition is met, it returns a passthrough handler without any CORS validation.
func SelectCORSMiddleware(
	allowedOrigins []string,
	allowAll, denyAll bool,
) func(http.Handler) http.Handler {
	if denyAll {
		return DenyAllOriginsMiddleware()
	}

	if allowAll {
		return CORSMiddleware()
	}

	if len(allowedOrigins) > 0 {
		return WhitelistCORSMiddleware(allowedOrigins)
	}

	// Neither allowAll, denyAll, nor allowedOrigins is set, so return the next handler without any CORS middleware.
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			zerologr.V(20).Info("CORS middleware: no CORS configuration set, skipping CORS headers")
			next.ServeHTTP(w, r)
		})
	}
}

func DenyAllOriginsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			zerologr.V(20).Info("CORS middleware: denying all origins")

			if HasOrigin(r) {
				apierror.ErrorHandler(w, r, apierror.ErrForbidden)
				return
			}

			zerologr.V(20).Info("CORS middleware: not a browser request, skipping")
			next.ServeHTTP(w, r)
		})
	}
}

// HasOrigin checks if the request has an Origin header.
func HasOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") != ""
}

// CORSMiddleware is a middleware that adds CORS headers to the response. Input Origin
// are mirrored back in the Access-Control-Allow-Origin header, allowing any origin to access the resource.
func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			zerologr.V(20).Info("CORS middleware: attaching CORS headers")
			if !HasOrigin(r) {
				// No Origin header present, so this is not a browser request.
				zerologr.V(20).Info("CORS middleware: not a browser request, skipping CORS headers")
				next.ServeHTTP(w, r)
				return
			}
			SetCORSHeaders(w, r.Header.Get("Origin"))

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// WhitelistCORSMiddleware is a middleware that adds CORS headers to the response for requests from allowed origins.
// Non-whitelisted origins will receive a 403 Forbidden response.
func WhitelistCORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			zerologr.V(20).Info("Whitelist CORS middleware: checking allowed origins")
			if !HasOrigin(r) {
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
					SetCORSHeaders(w, origin)

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
}

// SetCORSHeaders sets the necessary CORS headers on the response writer for the given origin.
func SetCORSHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set(
		"Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	)
	// Read more here for whitelisted headers: https://developer.mozilla.org/en-US/docs/Glossary/CORS-safelisted_response_header
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, "+CSRFTokenHeader)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}
