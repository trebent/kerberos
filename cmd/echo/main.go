// nolint
// echo is a simple HTTP server that echoes back the request
// body and headers. It is used to probe how Kerberos enriches requests prior to dispatch.
// echo will respond with what headers and request body was supplied to it, making it useful
// for debugging traces and any other headers that Kerberos may have added to the request.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

// response is the structure of the JSON response that will be sent back to the client from echo.
type response struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body,omitempty"`
}

var (
	_ io.Writer = &response{}

	verbosity = envparser.Register(&envparser.Opts[int]{
		Name:  "LOG_VERBOSITY",
		Desc:  "Sets the logging verbosity level.",
		Value: defaultLogVerbosity,
	})
	version = envparser.Register(&envparser.Opts[string]{
		Name:  "VERSION",
		Desc:  "Sets the application version.",
		Value: "unset",
	})
	port = envparser.Register(&envparser.Opts[int]{
		Name:  "PORT",
		Desc:  "Port to listen on.",
		Value: defaultPort,
	})
	observabilityEnabled = envparser.Register(&envparser.Opts[bool]{
		Name:  "OBSERVABILITY_ENABLED",
		Desc:  "Enables or disables observability features.",
		Value: true,
	})
	tlsCertFile = envparser.Register(&envparser.Opts[string]{
		Name: "TLS_CERT_FILE",
		Desc: "Path to the PEM-encoded server certificate file. When set together with TLS_KEY_FILE, the server enables TLS.",
	})
	tlsKeyFile = envparser.Register(&envparser.Opts[string]{
		Name: "TLS_KEY_FILE",
		Desc: "Path to the PEM-encoded server private key file. When set together with TLS_CERT_FILE, the server enables TLS.",
	})
	tlsClientCAFile = envparser.Register(&envparser.Opts[string]{
		Name: "TLS_CLIENT_CA_FILE",
		Desc: "Path to a PEM-encoded CA certificate bundle used to verify client certificates (mTLS). Requires TLS_CERT_FILE and TLS_KEY_FILE to be set.",
	})

	tracer = otel.GetTracerProvider().Tracer(tracerName)
)

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

	if observabilityEnabled.Value() {
		zerologr.Info("Initializing OpenTelemetry instrumentation")
		shutdown, err := intotel.Instrument(signalCtx, "echo", version.Value(), true)
		if err != nil {
			zerologr.Error(err, "Failed to initialize OpenTelemetry")
			os.Exit(1)
		}
		defer shutdown(context.Background())
	}

	// Create a new HTTP server
	srv := &http.Server{
		Addr:      fmt.Sprintf(":%d", port.Value()),
		TLSConfig: tlsConfig(),
	}

	// Register the echo handler
	http.HandleFunc("/", handler)

	// Start the server
	go func() {
		var err error
		if tlsCertFile.Value() != "" && tlsKeyFile.Value() != "" {
			err = srv.ListenAndServeTLS(tlsCertFile.Value(), tlsKeyFile.Value())
		} else {
			err = srv.ListenAndServe()
		}

		if !errors.Is(err, http.ErrServerClosed) {
			zerologr.Error(err, "Server start/stop failed")
		}
	}()

	zerologr.Info(
		"Echo server started",
		"port", port.Value(),
		"tlsEnabled", tlsCertFile.Value() != "" && tlsKeyFile.Value() != "",
		"observabilityEnabled", observabilityEnabled.Value(),
	)
	<-signalCtx.Done()
	srv.Shutdown(context.Background())
	zerologr.Info("Echo gracefully stopped")
}

// tlsConfig returns a TLS configuration if TLS_CERT_FILE and TLS_KEY_FILE are set, otherwise it returns nil. If TLS_CLIENT_CA_FILE is set, it configures the server for mutual TLS (mTLS) by requiring and verifying client certificates against the provided CA bundle.
func tlsConfig() *tls.Config {
	if tlsCertFile.Value() != "" && tlsKeyFile.Value() != "" {
		zerologr.Info(
			"TLS enabled",
			"certFile", tlsCertFile.Value(),
			"keyFile", tlsKeyFile.Value(),
		)

		tlsCfg := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		if caFile := tlsClientCAFile.Value(); caFile != "" {
			caPEM, err := os.ReadFile(caFile)
			if err != nil {
				zerologr.Error(err, "Failed to read client CA file")
				os.Exit(1)
			}
			caPool := x509.NewCertPool()
			if !caPool.AppendCertsFromPEM(caPEM) {
				zerologr.Error(errors.New("no valid certificates found in client CA file"), "Failed to load client CA")
				os.Exit(1)
			}
			tlsCfg.ClientCAs = caPool
			tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
		}
		return tlsCfg
	}

	return nil
}

// handler is the HTTP handler function for the echo server. It reads the request body and headers, and writes them back in the response as JSON.
func handler(w http.ResponseWriter, r *http.Request) {
	zerologr.Info(r.Method+" "+r.URL.String(), "size", r.ContentLength)

	for key, values := range r.Header {
		zerologr.Info("Header", key, fmt.Sprintf("%s", values))
	}

	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		zerologr.Info("Got valid trace span", "traceID", span.SpanContext().TraceID().String(), "spanID", span.SpanContext().SpanID().String())
	}

	_, newSpan := tracer.Start(ctx, "echoing", trace.WithSpanKind(trace.SpanKindServer))
	zerologr.Info("New span", "traceID", newSpan.SpanContext().TraceID().String(), "spanID", newSpan.SpanContext().SpanID().String())
	defer newSpan.End()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Echo-Server", "true")

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
}
