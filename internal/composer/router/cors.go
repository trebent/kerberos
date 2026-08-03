package router

import (
	"net/http"
	"slices"

	"github.com/trebent/kerberos/internal/config"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
)

func corsConfigured(backend *config.RouterBackend) bool {
	if backend.Origins == nil {
		return false
	}

	return backend.Origins.AllowAll ||
		backend.Origins.DenyAll ||
		len(backend.Origins.AllowedOrigins) > 0
}

func ValidateOrigin(origin string, originsCfg *config.Origins) error {
	if originsCfg.DenyAll {
		return apierror.New(http.StatusForbidden, "origin denied")
	}

	if originsCfg.AllowAll {
		return nil
	}

	if slices.Contains(originsCfg.AllowedOrigins, origin) {
		return nil
	}

	return apierror.New(http.StatusForbidden, "origin denied")
}
