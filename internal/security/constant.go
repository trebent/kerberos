package security

import (
	"time"
)

const (
	// CSRFTokenHeader is the name of the header used to send the CSRF token in requests.
	//nolint:gosec // really?
	CSRFTokenHeader = "X-Krb-Csrf-Token"

	SessionCookieName = "session"
	SessionMaxAge     = 15 * time.Minute
	RefreshCookieName = "refresh"
	RefreshMaxAge     = 1 * time.Hour
	CSRFCookieName    = "csrf"
	CSRFMaxAge        = RefreshMaxAge
)
