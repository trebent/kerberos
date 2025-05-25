package forwarder

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/kerberos/internal/router"
	"github.com/trebent/zerologr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var (
	forwardPattern = regexp.MustCompile(`^/gw/backend/[-_a-z0-9]+/(.+)?$`)

	ErrFailedPatternMatch = errors.New("forward pattern match failed")
	ErrFailedForwarding   = errors.New("failed to forward request")
)

const expectedPatternMatches = 2

// Forwarder returns a HTTP handler that forwards any received requests to
// their designated backends.
func Forwarder() http.Handler {
	return http.HandlerFunc(func(wrapped http.ResponseWriter, r *http.Request) {
		// Obtain matching backend to route to.
		// Forward request and pipe forwarded response into origin response.
		rLogger, _ := logr.FromContext(r.Context())
		rLogger = rLogger.WithName("forwarder")
		rLogger.Info("Forwarding request")

		backend := router.BackendFromContext(r.Context())

		forwardURL := forwardPattern.FindStringSubmatch(r.URL.Path)
		if len(forwardURL) < expectedPatternMatches {
			rLogger.Error(
				fmt.Errorf("%w: %s", ErrFailedPatternMatch, r.URL.Path),
				"Pattern match failed",
			)
			response.JSONError(wrapped, ErrFailedForwarding, http.StatusInternalServerError)
			return
		}

		forwardRequest, err := http.NewRequestWithContext(
			r.Context(),
			r.Method,
			fmt.Sprintf(
				"http://%s/%s",
				net.JoinHostPort(backend.Host(), strconv.Itoa(backend.Port())),
				forwardURL[1],
			),
			r.Body,
		)
		if err != nil {
			rLogger.Error(err, "Failed to create request")
			response.JSONError(wrapped, ErrFailedForwarding, http.StatusInternalServerError)
			return
		}

		forwardRequest.Header = r.Header
		otel.GetTextMapPropagator().
			Inject(r.Context(), propagation.HeaderCarrier(forwardRequest.Header))

		client := http.Client{}
		resp, err := client.Do(forwardRequest)
		if err != nil {
			rLogger.Error(err, "Failed to forward request")
			response.JSONError(wrapped, ErrFailedForwarding, http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			rLogger.V(100).Info("Adding header to response", "key", key, "values", values)
			for _, value := range values {
				wrapped.Header().Add(key, value)
			}
		}

		_, err = io.Copy(wrapped, resp.Body)
		if err != nil {
			rLogger.Error(err, "Failed to copy response body")
			response.JSONError(wrapped, ErrFailedForwarding, http.StatusInternalServerError)
			return
		}

		wrapped.WriteHeader(resp.StatusCode)
		rLogger.V(50).Info("Forwarded request")
	})
}

// Test is an endpoint to verify metric generation works as expected. Wanted
// status code is passed as a query parameter. Any body present in the request
// is echoed back.
func Test() http.Handler {
	return http.HandlerFunc(func(wrapped http.ResponseWriter, r *http.Request) {
		zerologr.Info("Received test request", "method", r.Method, "path", r.URL.Path)

		statusCode, err := func() (int, error) {
			queryParam := r.URL.Query().Get("status_code")
			if queryParam != "" {
				i, err := strconv.ParseInt(queryParam, 10, 16)
				return int(i), err
			}
			return http.StatusOK, nil
		}()

		if err != nil {
			wrapped.WriteHeader(http.StatusInternalServerError)
			zerologr.Error(err, "Failed to decode the status_code query parameter")
			return
		}

		// nolint: govet
		if _, err := io.Copy(wrapped, r.Body); err != nil {
			wrapped.WriteHeader(http.StatusInternalServerError)
			zerologr.Error(err, "Failed to write request body into response body")
			return
		}

		zerologr.Info("Responding with status code", "status_code", statusCode)
		wrapped.WriteHeader(statusCode)
	})
}
