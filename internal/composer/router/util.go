package router

import (
	"fmt"
	"net/http"

	apierror "github.com/trebent/kerberos/internal/oapi/error"
)

//nolint:errname // this is on purpose
var apiErrBadRequest = apierror.New(
	http.StatusBadRequest,
	"failed to extract backend name from request path",
)

// GetBackendName extracts the backend name from the request URL path.
func GetBackendName(req *http.Request) (string, error) {
	reqPath := routePattern.FindStringSubmatch(req.URL.Path)

	if len(reqPath) < expectedPatternMatches {
		return "", fmt.Errorf("%w: %s", apiErrBadRequest, req.URL.Path)
	}

	return reqPath[1], nil
}
