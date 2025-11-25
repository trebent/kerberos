// nolint
package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	obs "github.com/trebent/kerberos/internal/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	shutdown, _ := obs.Instrument(context.TODO(), "tracing-testing", "0.1.0")
	defer shutdown(context.Background())
	println("Starting server on :15000")

	signalCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	server := http.Server{
		Addr: ":15000",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			println("Received request:", r.Method, r.URL.String())

			for k, v := range r.Header {
				println(" Header:", k, "Value:", v[0])
			}

			ctx := otel.GetTextMapPropagator().
				Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			span := trace.SpanFromContext(ctx)

			if span.IsRecording() {
				println("Span is recording")
			}
			// Any modification of the parent span (from the client) won't have any effect, just for show.
			span.SetAttributes(attribute.String("server-attribute", "value"))

			if span.SpanContext().IsValid() {
				println("Span is valid")
			}

			// Now for a real child span, linked to the client span since ctx is extracted using the text map propagator.
			_, childSpan := otel.Tracer("server").
				Start(ctx, "server-request", trace.WithSpanKind(trace.SpanKindServer))
			defer childSpan.End()
			// This will show in jaeger.
			childSpan.SetAttributes(attribute.String("server-attribute", "value"))

			// This does not do anythign, just for show.
			span.SetAttributes(
				attribute.KeyValue{
					Key:   "server-attribute",
					Value: attribute.StringValue(time.Now().String()),
				},
			)
			// This does not do anythign, just for show.
			span.End()

			w.WriteHeader(http.StatusOK)
		}),
	}
	go server.ListenAndServe()

	<-signalCtx.Done()

	server.Shutdown(context.Background())
	cancel()
}
