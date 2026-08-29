package otel

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware returns an HTTP middleware that adds an OpenTelemetry trace to the request context.
func TracingMiddleware(scopeName, version string, next http.Handler) http.Handler {
	tracer := otel.Tracer(scopeName, trace.WithInstrumentationVersion(version))
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := otel.GetTextMapPropagator().
			Extract(req.Context(), propagation.HeaderCarrier(req.Header))
		ctx, span := tracer.Start(
			ctx,
			fmt.Sprintf("%s %s", req.Method, req.URL.String()),
			spanStartOpts(req)...,
		)
		defer span.End()

		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func spanStartOpts(req *http.Request) []trace.SpanStartOption {
	opts := make([]trace.SpanStartOption, 3)
	opts[0] = trace.WithSpanKind(trace.SpanKindServer)
	opts[1] = trace.WithAttributes(semconv.HTTPMethod(req.Method))
	opts[2] = trace.WithAttributes(semconv.HTTPURL(req.URL.String()))

	return opts
}

func MetricsMiddleware(metrics *StdHTTPMetrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, req)
		metrics.Bump(req.Context(), w, req.Body, req, time.Since(start))
	})
}
