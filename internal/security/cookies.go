package security

import "net/http"

// StripKerberosCookies removes the Kerberos session, refresh and CSRF cookies from the request,
// so they are not sent on to backends or proxied tools. Other cookies are kept.
func StripKerberosCookies(r *http.Request) {
	cookies := r.Cookies()
	r.Header.Del("Cookie")

	for _, c := range cookies {
		switch c.Name {
		case SessionCookieName, RefreshCookieName, CSRFCookieName:
			continue
		}

		r.AddCookie(c)
	}
}
