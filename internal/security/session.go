package security

import "net/http"

// SessionCookieString returns a string representation of a session cookie with the given value, SameSite attribute, and domain.
func SessionCookieString(
	value string,
	sameSite http.SameSite,
	domain string,
) string {
	c := SessionCookie(value, sameSite, domain)
	return c.String()
}

// SessionCookie returns an http.Cookie struct representing a session cookie with the given value, SameSite attribute, and domain.
func SessionCookie(
	value string,
	sameSite http.SameSite,
	domain string,
) http.Cookie {
	//nolint:gosec // SameSite configurable
	return http.Cookie{
		Name:     SessionCookieName,
		Value:    value,
		SameSite: sameSite,
		HttpOnly: true,
		Secure:   true,
		Domain:   domain,
		MaxAge:   int(SessionMaxAge.Seconds()),
	}
}

// ExpiredSessionCookieString returns a string representation of an expired session cookie with the given SameSite attribute and domain.
func ExpiredSessionCookieString(
	sameSite http.SameSite,
	domain string,
) string {
	c := ExpiredSessionCookie(sameSite, domain)
	return c.String()
}

// ExpiredSessionCookie returns an http.Cookie struct representing an expired session cookie with the given SameSite attribute and domain.
func ExpiredSessionCookie(
	sameSite http.SameSite,
	domain string,
) http.Cookie {
	//nolint:gosec // SameSite configurable
	return http.Cookie{
		Name:     SessionCookieName,
		Value:    "expired",
		SameSite: sameSite,
		HttpOnly: true,
		Secure:   true,
		Domain:   domain,
		MaxAge:   -1,
	}
}

// RefreshCookieString returns a string representation of a refresh cookie with the given value, SameSite attribute, domain, and path.
func RefreshCookieString(
	value string,
	sameSite http.SameSite,
	domain string,
	path string,
) string {
	c := RefreshCookie(value, sameSite, domain, path)
	return c.String()
}

// RefreshCookie returns an http.Cookie struct representing a refresh cookie with the given value, SameSite attribute, domain, and path.
func RefreshCookie(
	value string,
	sameSite http.SameSite,
	domain string,
	path string,
) http.Cookie {
	//nolint:gosec // SameSite configurable
	return http.Cookie{
		Name:     RefreshCookieName,
		Value:    value,
		SameSite: sameSite,
		HttpOnly: true,
		Secure:   true,
		Domain:   domain,
		Path:     path,
		MaxAge:   int(RefreshMaxAge.Seconds()),
	}
}

// ExpiredRefreshCookieString returns a string representation of an expired refresh cookie with the given SameSite attribute, domain, and path.
func ExpiredRefreshCookieString(
	sameSite http.SameSite,
	domain string,
	path string,
) string {
	c := ExpiredRefreshCookie(sameSite, domain, path)
	return c.String()
}

// ExpiredRefreshCookie returns an http.Cookie struct representing an expired refresh cookie with the given SameSite attribute, domain, and path.
func ExpiredRefreshCookie(
	sameSite http.SameSite,
	domain string,
	path string,
) http.Cookie {
	//nolint:gosec // SameSite configurable
	return http.Cookie{
		Name:     RefreshCookieName,
		Value:    "expired",
		SameSite: sameSite,
		HttpOnly: true,
		Secure:   true,
		Domain:   domain,
		Path:     path,
		MaxAge:   -1,
	}
}
