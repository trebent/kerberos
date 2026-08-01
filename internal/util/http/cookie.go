package http

import (
	"net/http"

	"github.com/trebent/zerologr"
)

// ConvertSameSite converts a string representation of SameSite to the corresponding http.SameSite value.
// If the input string is invalid, it logs a warning and returns http.SameSiteDefaultMode.
// Valid inputs are "Lax", "Strict", and "None".
func ConvertSameSite(sameSite string) http.SameSite {
	switch sameSite {
	case "Lax":
		return http.SameSiteLaxMode
	case "Strict":
		return http.SameSiteStrictMode
	case "None":
		return http.SameSiteNoneMode
	default:
		zerologr.Info("WARN: invalid input SameSite, using default")
		return http.SameSiteDefaultMode
	}
}
