package router

import (
	"fmt"
	"net/http"
	"regexp"

	apierror "github.com/trebent/kerberos/internal/oapi/error"
)

var (
	routePattern = regexp.MustCompile(`^/gw/backend/([-_a-z0-9]+)?/.*$`)

	//nolint:errname // this is on purpose
	apiErrBadRequest = apierror.New(
		http.StatusBadRequest,
		"failed to extract backend name from request path",
	)
)

// GetBackendName extracts the backend name from the request URL path.
func GetBackendName(req *http.Request) (string, error) {
	reqPath := routePattern.FindStringSubmatch(req.URL.Path)

	if len(reqPath) < expectedPatternMatches {
		return "", fmt.Errorf("%w: %s", apiErrBadRequest, req.URL.Path)
	}

	if reqPath[1] == "" {
		return "", fmt.Errorf("%w: %s", apiErrBadRequest, req.URL.Path)
	}

	return reqPath[1], nil
}
