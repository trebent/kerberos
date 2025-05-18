package otel

import (
	"net/http"

	"github.com/go-logr/logr"
)

func Middleware(next http.Handler, logger logr.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info(r.Method + " " + r.URL.Path)

		next.ServeHTTP(w, r)

		logger.Info(r.Method + " " + r.URL.Path)
	})
}
