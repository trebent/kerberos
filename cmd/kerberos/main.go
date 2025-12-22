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

	"github.com/prometheus/common/version"
	"github.com/trebent/envparser"
	composerctx "github.com/trebent/kerberos/internal/composer/context"
	"github.com/trebent/kerberos/internal/composer/forwarder"
	obs "github.com/trebent/kerberos/internal/composer/observability"
	"github.com/trebent/kerberos/internal/composer/router"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/env"
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

	cfg := setupConfig()

	// Set up monitoring
	zerologr.Set(zerologr.New(&zerologr.Opts{
		Console: env.LogToConsole.Value(),
		Caller:  true,
		V:       env.LogVerbosity.Value(),
	}).
		WithValues(string(semconv.ServiceNameKey), serviceName, string(semconv.ServiceVersionKey), env.Version.Value()).
		WithName("krb"),
	)
	startLogger := zerologr.WithName("start")
	startLogger.Info("Starting Kerberos API GW server", "port", env.Port.Value())

	signalCtx, signalCancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer signalCancel()

	cleanup, err := obs.Instrument(signalCtx, serviceName, version.Version)
	if err != nil {
		startLogger.Error(err, "Failed to instrument OpenTelemetry")
		os.Exit(1) // nolint: gocritic
	}
	defer cleanup(context.Background()) // nolint: errcheck

	// Start Kerberos API GW server
	// nolint: govet
	if err := startServer(signalCtx, cfg); !errors.Is(err, http.ErrServerClosed) {
		startLogger.Error(err, "Kerberos HTTP server failed")
		os.Exit(1)
	}
	startLogger.Info("Kerberos API GW server stopped")
}

// setupConfig sets up the configuration map and registers all necessary
// configurations. It returns the configuration map after calling Parse().
func setupConfig() config.Map {
	cfg := config.New()

	var (
		err              error
		obsConfigName    string
		routerConfigName string
	)

	// Register all configurations.
	obsConfigName, err = obs.RegisterWith(cfg)
	must(err)
	routerConfigName, err = router.RegisterWith(cfg)
	must(err)

	// Load all input configuration data.
	if env.ObsJSONFile.Value() != "" {
		obsData, _ := os.ReadFile(env.ObsJSONFile.Value())
		cfg.MustLoad(obsConfigName, obsData)
	}

	routerData, _ := os.ReadFile(env.RouteJSONFile.Value())
	cfg.MustLoad(routerConfigName, routerData)

	// Parse configurations.
	//nolint: govet
	if err := cfg.Parse(); err != nil {
		zerologr.Error(err, "Failed to parse configurations")
		os.Exit(1) // nolint: gocritic
	}

	zerologr.Info("Loaded configurations")

	return cfg
}

// startServer starts the HTTP server and listens for incoming requests.
// It returns an error if the server fails to start and when stopping. If
// the server is stopped, it returns http.ErrServerClosed.
func startServer(ctx context.Context, cfg config.Map) error {
	mux := http.NewServeMux()

	// OTEL middleware must be called first, then forwarding can happen.
	//
	// start tracing/metrics/logs
	// resp = forward(request)
	// stop tracing/metrics/logs
	if env.TestEndpoint.Value() {
		testHandler := obs.Middleware(forwarder.Test())

		mux.Handle("/test", testHandler)
		zerologr.Info("Test endpoint enabled")
	}

	zerologr.Info("Loading router")

	// This is the main endpoint for the API GW. Every incoming request passes through the backend
	// middleware for backend detection, and the route middleware for route matching. A failure to
	// match a backend yields a 404. Route matching is optional, it's use is limited to enrich
	// metrics with route information which can't be derived from the raw URL (as the metric
	// dimensions would grow out of control).
	r := router.New(
		&router.Opts{Cfg: cfg},
	)
	zerologr.Info("Router loaded")

	gwHandler := obs.Middleware(
		router.Middleware(forwarder.Forwarder(composerctx.TargetContextKey), r),
	)
	mux.Handle("/gw/", gwHandler)

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

func must(err error) {
	if err != nil {
		panic(err)
	}
}
