package security

import (
	"net/http"
	"strings"
)

// StripKerberosCookies removes the Kerberos session, refresh and CSRF cookies from the Cookie
// header, so they are not sent on to backends or proxied tools. Other cookies are kept as they
// are, and the header is removed when no cookies are left.
func StripKerberosCookies(h http.Header) {
	lines := h.Values("Cookie")
	if len(lines) == 0 {
		return
	}

	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		for part := range strings.SplitSeq(line, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			name, _, _ := strings.Cut(part, "=")
			switch strings.TrimSpace(name) {
			case SessionCookieName, RefreshCookieName, CSRFCookieName:
				continue
			}

			kept = append(kept, part)
		}
	}

	if len(kept) == 0 {
		h.Del("Cookie")
		return
	}

	h.Set("Cookie", strings.Join(kept, "; "))
}
