package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/trebent/envparser"
	"github.com/trebent/kerberos/internal/env"
	"github.com/trebent/kerberos/internal/otel"
	"github.com/trebent/zerologr"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// nolint: gochecknoglobals
var (
	readTimeout  time.Duration
	writeTimeout time.Duration
)

// TODO: Add support for prefix exemptions to allow OTEL vars to be set without
// the KRB prefix.
// //nolint:gochecknoinits
// func init() {
// 	envparser.Prefix = "KRB" //nolint:reassign
// }

func main() {
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = func() { //nolint:reassign
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.CommandLine.PrintDefaults()
		fmt.Fprint(flag.CommandLine.Output(), "\n")
		fmt.Fprint(flag.CommandLine.Output(), envparser.Help())
	}

	flag.Parse()
	// ExitOnError = true
	_ = env.Parse()

	readTimeout = time.Duration(env.ReadTimeoutSeconds.Value()) * time.Second
	writeTimeout = time.Duration(env.WriteTimeoutSeconds.Value()) * time.Second

	// Set up monitoring
	logger := zerologr.New(&zerologr.Opts{
		Console: env.LogToConsole.Value(),
		Caller:  true,
		V:       env.LogVerbosity.Value(),
	})
	zerologr.Set(logger.WithName("global"))
	logger = logger.WithName("start")
	logger.Info("Starting Kerberos API GW server", "port", env.Port.Value())

	signalCtx, signalCancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer signalCancel()

	shutdown, err := otel.Instrument(signalCtx)
	if err != nil {
		logger.Error(err, "Failed to instrument OpenTelemetry")
		os.Exit(1) // nolint: gocritic
	}
	defer shutdown(context.Background()) // nolint: errcheck

	// Start Kerberos API GW server
	if err := startServer(signalCtx); !errors.Is(err, http.ErrServerClosed) { // nolint: govet
		logger.Error(err, "Failed to start Kerberos HTTP server")
		os.Exit(1)
	}
	logger.Info("Kerberos API GW server stopped")
}

// startServer starts the HTTP server and listens for incoming requests.
// It returns an error if the server fails to start and when stopping. If
// the server is stopped, it returns http.ErrServerClosed.
func startServer(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		zerologr.Info("Received request", "method", r.Method, "path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})
	if env.TestEndpoint.Value() {
		mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			zerologr.Info("Received test request", "method", r.Method, "path", r.URL.Path)

			statusCode, err := func() (int, error) {
				queryParam := r.URL.Query().Get("status_code")
				if queryParam != "" {
					i, err := strconv.ParseInt(queryParam, 10, 16)
					return int(i), err
				}
				return 200, nil
			}()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				zerologr.Error(err, "Failed to decode the status_code query parameter")
				return
			}
			zerologr.Info("Responding with status code", "status_code", statusCode)
			w.WriteHeader(statusCode)
		})
		zerologr.Info("Test endpoint enabled")
	}
	handler := otelhttp.NewHandler(mux, "krb")

	server := http.Server{
		Addr:         fmt.Sprintf(":%d", env.Port.Value()),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		Handler:      handler,
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.ListenAndServe()
	}()

	var (
		srvErr      error
		shutdownErr error
	)
	select {
	case <-ctx.Done():
		zerologr.Info("Stopping server")

		timeoutCtx, timeoutCancel := context.WithTimeout(
			context.Background(),
			readTimeout+writeTimeout,
		)
		defer timeoutCancel()

		shutdownErr = server.Shutdown(timeoutCtx)
		if shutdownErr != nil {
			zerologr.Error(shutdownErr, "Server shutdown error")
		}
		srvErr = <-errChan
	case srvErr = <-errChan:
		zerologr.Error(srvErr, "Server start error")
	}

	return errors.Join(srvErr, shutdownErr)
}
