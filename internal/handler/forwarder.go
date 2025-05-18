package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/trebent/zerologr"
)

// Forwarder returns a HTTP handler that forwards any received requests to
// their designated backends, if a matching one is found.
func Forwarder(_ logr.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Obtain matching backend to route to.
		// Forward request and pipe forwarded response into origin response.
	})
}

// Use this endpoint to verify metric generation works as expected. Wanted
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

		if _, err := io.Copy(w, r.Body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			zerologr.Error(err, "Failed to write request body into response body")
			return
		}

		zerologr.Info("Responding with status code", "status_code", statusCode)
		w.WriteHeader(statusCode)
	})
}
