// nolint
// echo is a simple HTTP server that echoes back the request
// body and headers. It is used for testing purposes.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/trebent/envparser"
	intotel "github.com/trebent/kerberos/internal/otel"
	"github.com/trebent/zerologr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

type response struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body,omitempty"`
}

var _ io.Writer = &response{}

const (
	tracerName          = "echo"
	defaultLogVerbosity = 0
	defaultPort         = 15000
)

// Write implements io.Writer.
func (r *response) Write(p []byte) (n int, err error) {
	r.Body = append(r.Body, p...)
	return len(p), nil
}

func main() {
	verbosity := envparser.Register(&envparser.Opts[int]{
		Name:  "LOG_VERBOSITY",
		Desc:  "Sets the logging verbosity level.",
		Value: defaultLogVerbosity,
	})
	version := envparser.Register(&envparser.Opts[string]{
		Name:  "VERSION",
		Desc:  "Sets the application version.",
		Value: "unset",
	})
	port := envparser.Register(&envparser.Opts[int]{
		Name:  "PORT",
		Desc:  "Port to listen on.",
		Value: defaultPort,
	})

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "An echo HTTP server that echoes back request body and headers.\n\n")
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), envparser.Help())
	}
	flag.Parse()
	envparser.Parse()

	logger := zerologr.New(&zerologr.Opts{Console: true, Caller: true, V: verbosity.Value()})
	logger.WithName("echo")
	logger.WithValues(semconv.ServiceName("echo"), semconv.ServiceVersion(version.Value()))
	zerologr.Set(logger)

	signalCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	shutdown, err := intotel.Instrument(signalCtx, "echo", "0.1.0")
	if err != nil {
		zerologr.Error(err, "Failed to initialize OpenTelemetry")
		os.Exit(1)
	}
	defer shutdown(context.Background())

	// Create a new HTTP server
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", port.Value()),
	}

	// Register the echo handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		zerologr.Info(r.Method+" "+r.URL.String(), "size", r.ContentLength)

		for key, values := range r.Header {
			zerologr.Info("Header", key, fmt.Sprintf("%s", values))
		}

		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			zerologr.Info("Got valid trace span", "traceID", span.SpanContext().TraceID().String(), "spanID", span.SpanContext().SpanID().String())
		}
		tracer := newTracer(otel.GetTracerProvider())

		_, newSpan := tracer.Start(ctx, "echoing", trace.WithSpanKind(trace.SpanKindServer))
		zerologr.Info("New span", "traceID", newSpan.SpanContext().TraceID().String(), "spanID", newSpan.SpanContext().SpanID().String())
		defer newSpan.End()

		w.Header().Set("Content-Type", "application/json")

		resp := &response{
			Method:  r.Method,
			URL:     r.URL.String(),
			Headers: r.Header,
		}

		if r.Body != nil && r.Body != http.NoBody {
			defer r.Body.Close()

			data, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(
					w,
					"{\"error\": \"failed to read request body\"}",
					http.StatusInternalServerError,
				)
				return
			}
			zerologr.V(20).Info("Read body: "+string(data), "size", len(data))

			resp.Body = data
		}

		responseBytes, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			http.Error(
				w,
				"{\"error\": \"failed to marshal response\"}",
				http.StatusInternalServerError,
			)
			return
		}

		zerologr.V(20).Info("Writing response: "+string(responseBytes), "size", len(responseBytes))

		_, _ = w.Write(responseBytes)
	})

	go func() {
		// Start the server
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			zerologr.Error(err, "Server start/stop failed: %v")
		}
	}()

	zerologr.Info("Echo server started", "port", port.Value())
	<-signalCtx.Done()
	srv.Shutdown(context.Background())
	zerologr.Info("Echo gracefully stopped")
}

func newTracer(provider trace.TracerProvider) trace.Tracer {
	return provider.Tracer(tracerName)
}
