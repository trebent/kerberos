package security

const (
	// CSRFTokenHeader is the name of the header used to send the CSRF token in requests.
	//nolint:gosec // really?
	CSRFTokenHeader = "X-Krb-Csrf-Token"

	SessionCookieName = "session"
	CSRFCookieName    = "csrf"
)
