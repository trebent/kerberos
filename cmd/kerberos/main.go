package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/trebent/envparser"
	"github.com/trebent/kerberos/internal/env"
	krbhandler "github.com/trebent/kerberos/internal/handler"
	krbotel "github.com/trebent/kerberos/internal/otel"
	"github.com/trebent/kerberos/internal/version"
	"github.com/trebent/zerologr"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// nolint: gochecknoglobals
var (
	readTimeout  time.Duration
	writeTimeout time.Duration
)

const serviceName = "krb"

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
	rootLogger := zerologr.New(&zerologr.Opts{
		Console: env.LogToConsole.Value(),
		Caller:  true,
		V:       env.LogVerbosity.Value(),
	}).WithValues(string(semconv.ServiceNameKey), serviceName, string(semconv.ServiceVersionKey), version.Version())
	zerologr.Set(rootLogger.WithName("global"))
	startLogger := rootLogger.WithName("start")
	startLogger.Info("Starting Kerberos API GW server", "port", env.Port.Value())

	signalCtx, signalCancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer signalCancel()

	shutdown, err := krbotel.Instrument(signalCtx, serviceName, version.Version())
	if err != nil {
		startLogger.Error(err, "Failed to instrument OpenTelemetry")
		os.Exit(1) // nolint: gocritic
	}
	defer shutdown(context.Background()) // nolint: errcheck

	// Start Kerberos API GW server
	// nolint: govet
	if err := startServer(signalCtx, rootLogger); !errors.Is(err, http.ErrServerClosed) {
		startLogger.Error(err, "Failed to start Kerberos HTTP server")
		os.Exit(1)
	}
	startLogger.Info("Kerberos API GW server stopped")
}

// startServer starts the HTTP server and listens for incoming requests.
// It returns an error if the server fails to start and when stopping. If
// the server is stopped, it returns http.ErrServerClosed.
func startServer(ctx context.Context, rootLogger logr.Logger) error {
	mux := http.NewServeMux()

	// OTEL middleware must be called first, then forwarding can happen.
	//
	// start tracing/metrics/logs
	// resp = forward(request)
	// stop tracing/metrics/logs
	if env.TestEndpoint.Value() {
		mux.Handle("/test", krbotel.Middleware(
			krbhandler.Test(),
			rootLogger.WithName("request")),
		)
		zerologr.Info("Test endpoint enabled")
	}
	mux.Handle("/gw", krbotel.Middleware(
		krbhandler.Forwarder(rootLogger.WithName("forwarder")),
		rootLogger.WithName("request")),
	)

	server := http.Server{
		Addr:         fmt.Sprintf(":%d", env.Port.Value()),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		Handler:      mux,
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
