package security

import (
	"net/http"
	"strings"

	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/zerologr"
)

// CSRFMiddlewareWithExemptions is an HTTP middleware that checks for the presence of a valid CSRF token in requests,
// with the ability to exempt certain request paths from CSRF protection.
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
			zerologr.Error(err, "CSRF token validation failed")
			apierror.ErrorHandler(w, r, apierror.ErrForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// CSRFCookieString creates a new CSRF cookie with the given value, SameSite attribute, and domain, and returns its string representation.
func CSRFCookieString(
	value string,
	sameSite http.SameSite,
	domain string,
) string {
	c := CSRFCookie(value, sameSite, domain)
	return c.String()
}

// CSRFCookie creates a new CSRF cookie with the given value, SameSite attribute, and domain.
//
//nolint:gosec // HttpOnly false due to double-submit method requiring the cookie to be accessible by JavaScript.
func CSRFCookie(
	value string,
	sameSite http.SameSite,
	domain string,
) http.Cookie {
	return http.Cookie{
		Name:     CSRFCookieName,
		Value:    value,
		SameSite: sameSite,
		HttpOnly: false, // Double-submit method requires the cookie to be accessible by JavaScript
		Secure:   true,
		Domain:   domain,
		MaxAge:   int(RefreshMaxAge.Seconds()),
	}
}
