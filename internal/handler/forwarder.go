package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/kerberos/internal/router"
	"github.com/trebent/zerologr"
)

var forwardPattern = regexp.MustCompile(`^/gw/backend/.*/(.*)?$`)

// Forwarder returns a HTTP handler that forwards any received requests to
// their designated backends.
func Forwarder() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Obtain matching backend to route to.
		// Forward request and pipe forwarded response into origin response.
		rLogger, _ := logr.FromContext(r.Context())
		rLogger = rLogger.WithName("forwarder")
		rLogger.Info("Forwarding request")

		backend := router.BackendFromContext(r.Context())

		forwardURL := forwardPattern.FindStringSubmatch(r.URL.Path)
		if len(forwardURL) < 2 {
			rLogger.Error(errors.New("failed to match forward pattern"), "Pattern match failed")
			jsonError, _ := response.JSONError("forwarding failure")
			http.Error(w, string(jsonError), http.StatusInternalServerError)
			return
		}

		forwardRequest, err := http.NewRequest(
			r.Method,
			fmt.Sprintf("http://%s:%d/%s", backend.Host(), backend.Port(), forwardURL[1]),
			r.Body,
		)
		if err != nil {
			rLogger.Error(err, "Failed to create request")
			jsonError, _ := response.JSONError("forwarding failure")
			http.Error(w, string(jsonError), http.StatusInternalServerError)
			return
		}

		forwardRequest.Header = r.Header

		client := http.Client{}
		resp, err := client.Do(forwardRequest)
		if err != nil {
			rLogger.Error(err, "Failed to forward request")
			jsonError, _ := response.JSONError("forwarding failure")
			http.Error(w, string(jsonError), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			rLogger.Error(err, "Failed to copy response body")
			jsonError, _ := response.JSONError("forwarding failure")
			http.Error(w, string(jsonError), http.StatusInternalServerError)
			return
		}

		for key, values := range resp.Header {
			rLogger.Info("Adding header to response", "key", key, "values", values)
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(resp.StatusCode)
	})
}

// Test is an endpoint to verify metric generation works as expected. Wanted
// status code is passed as a query parameter. Any body present in the request
// is echoed back.
func Test() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			w.WriteHeader(http.StatusInternalServerError)
			zerologr.Error(err, "Failed to decode the status_code query parameter")
			return
		}

		// nolint: govet
		if _, err := io.Copy(w, r.Body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			zerologr.Error(err, "Failed to write request body into response body")
			return
		}

		zerologr.Info("Responding with status code", "status_code", statusCode)
		w.WriteHeader(statusCode)
	})
}
