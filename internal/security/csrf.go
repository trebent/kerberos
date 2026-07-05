package security

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/zerologr"
)

// GetCSRFToken generates a new CSRF token and returns it as an HTTP cookie.
func GetCSRFToken(maxAgeSeconds int) *http.Cookie {
	//nolint:gosec // on purpose
	return &http.Cookie{
		Name:  CSRFCookieName,
		Value: uuid.New().String(),
		// The double-submit method used by KRB means we need to inject the cookie value into
		// the KRB CSRF token header on the client side, so we cannot set the HttpOnly flag here.
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		Path:     "/",
		MaxAge:   maxAgeSeconds,
	}
}

func CSRFMiddlewareWithExemptions(exemptSuffixes []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		csrfProtected := CSRFMiddleware(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			zerologr.V(20).Info("CSRF middleware: checking request for CSRF token")

			for _, suffix := range exemptSuffixes {
				if strings.HasSuffix(r.URL.Path, suffix) {
					zerologr.V(20).Info(
						"CSRF middleware: request path is exempt from CSRF protection, skipping CSRF check",
					)
					next.ServeHTTP(w, r)
					return
				}
			}
			csrfProtected.ServeHTTP(w, r)
		})
	}
}

// CSRFMiddleware is an HTTP middleware that checks for the presence of a valid CSRF token in requests.
// It should be used for all endpoints that modify state (e.g., POST, PUT, DELETE).
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == "" {
			// No Origin header present, so this is not a browser request.
			zerologr.V(20).Info("CSRF middleware: not a browser request, skipping CSRF check")
			next.ServeHTTP(w, r)
			return
		}

		// Check if the request method is safe (GET, HEAD, OPTIONS, TRACE)
		if r.Method == http.MethodGet ||
			r.Method == http.MethodHead ||
			r.Method == http.MethodOptions ||
			r.Method == http.MethodTrace {
			next.ServeHTTP(w, r)
			return
		}

		// For unsafe methods (POST, PUT, DELETE), check for CSRF token
		csrfToken := r.Header.Get(CSRFTokenHeader)
		csrfCookie, err := r.Cookie(CSRFCookieName)

		if err != nil || csrfToken == "" || csrfCookie == nil || csrfToken != csrfCookie.Value {
			zerologr.Error(nil, "CSRF token validation failed")
			apierror.ErrorHandler(w, r, apierror.ErrForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
