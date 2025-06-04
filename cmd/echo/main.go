// nolint
// echo is a simple HTTP server that echoes back the request
// body and headers. It is used for testing purposes.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	krbotel "github.com/trebent/kerberos/internal/otel"
	"github.com/trebent/zerologr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type response struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
}

var _ io.Writer = &response{}

// Write implements io.Writer.
func (r *response) Write(p []byte) (n int, err error) {
	r.Body = append(r.Body, p...)
	return len(p), nil
}

func main() {
	signalCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := zerologr.New(&zerologr.Opts{Console: true}).WithName("echo")
	zerologr.Set(logger)

	shutdown, err := krbotel.Instrument(signalCtx, "echo", "0.1.0")
	if err != nil {
		zerologr.Error(err, "Failed to initialize OpenTelemetry")
		os.Exit(1)
	}
	defer shutdown(context.Background())

	// Create a new HTTP server
	srv := &http.Server{
		Addr: ":15000",
	}

	// Register the echo handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		println(r.Method+" "+r.URL.String(), "size", r.ContentLength)

		for key, values := range r.Header {
			println("  ", key, fmt.Sprintf("%s", values))
		}

		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
			println("Got valid trace span:", span.SpanContext().TraceID().String(), "with parent", span.SpanContext().SpanID().String())
		}

		w.Header().Set("Content-Type", "application/json")

		resp := &response{
			Method:  r.Method,
			URL:     r.URL.String(),
			Headers: r.Header,
		}

		if r.Body != nil && r.Body != http.NoBody {
			defer r.Body.Close()
			// Read the request body
			_, err := io.Copy(resp, r.Body)
			if err != nil {
				http.Error(
					w,
					"{\"error\": \"failed to read request body\"}",
					http.StatusInternalServerError,
				)
				return
			}
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

		_, _ = w.Write(responseBytes)
	})

	go func() {
		// Start the server
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-signalCtx.Done()
	srv.Shutdown(context.Background())
	log.Println("echo gracefully stopped")
}
