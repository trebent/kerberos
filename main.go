package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/trebent/envparser"
	"github.com/trebent/kerberos/internal/admin"
	"github.com/trebent/kerberos/internal/auth"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/custom"
	"github.com/trebent/kerberos/internal/composer/forwarder"
	obs "github.com/trebent/kerberos/internal/composer/observability"
	"github.com/trebent/kerberos/internal/composer/router"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/db/sqlite"
	internalenv "github.com/trebent/kerberos/internal/env"
	"github.com/trebent/kerberos/internal/oas"
	"github.com/trebent/zerologr"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// nolint: gochecknoglobals
var (
	readTimeout  time.Duration
	writeTimeout time.Duration

	configPath string
)

const serviceName = "krb"

func main() {
	flag.StringVar(&configPath, "config", "", "Path to the Kerberos configuration file (required).")
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = func() { //nolint:reassign
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.CommandLine.PrintDefaults()
		fmt.Fprint(flag.CommandLine.Output(), "\n")
		fmt.Fprint(flag.CommandLine.Output(), envparser.Help())
	}

	flag.Parse()

	if configPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --config flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// ExitOnError = true
	_ = internalenv.Parse()

	readTimeout = time.Duration(internalenv.ReadTimeoutSeconds.Value()) * time.Second
	writeTimeout = time.Duration(internalenv.WriteTimeoutSeconds.Value()) * time.Second

	// Set up monitoring
	zerologr.Set(zerologr.New(&zerologr.Opts{
		Console: internalenv.LogToConsole.Value(),
		Caller:  true,
		V:       internalenv.LogVerbosity.Value(),
	}).
		WithValues(string(semconv.ServiceNameKey), serviceName, string(semconv.ServiceVersionKey), internalenv.Version.Value()).
		WithName("krb"),
	)
	cfg := setupConfig()

	startLogger := zerologr.WithName("start")
	startLogger.Info("Starting Kerberos API GW server", "port", internalenv.Port.Value())

	signalCtx, signalCancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer signalCancel()

	cleanup, err := obs.Instrument(
		signalCtx,
		cfg.ObservabilityConfig,
		serviceName,
		internalenv.Version.Value(),
	)
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
func setupConfig() *config.RootConfig {
	zerologr.Info("Setting up configuration...")
	cfg := config.New()

	zerologr.Info("Loading configuration data")
	data, err := os.ReadFile(configPath)
	if err != nil {
		zerologr.Error(err, "Failed to read configuration file")
		os.Exit(1) // nolint: gocritic
	}
	cfg.Load(data)

	zerologr.Info("Configuration data loaded")

	zerologr.Info("Parsing configurations...")

	// Parse configurations.
	//nolint: govet
	if err := cfg.Parse(); err != nil {
		zerologr.Error(err, "Failed to parse configurations")
		os.Exit(1) // nolint: gocritic
	}

	zerologr.Info("Configuration parsed successfully")

	return cfg
}

// startServer starts the HTTP server and listens for incoming requests.
// It returns an error if the server fails to start and when stopping. If
// the server is stopped, it returns http.ErrServerClosed.
// nolint: funlen // welp
func startServer(ctx context.Context, cfg *config.RootConfig) error {
	mux := http.NewServeMux()
	db := sqlite.New(
		&sqlite.Opts{DSN: filepath.Join(internalenv.DBDirectory.Value(), sqlite.DBName)},
	)

	// Even though the admin configuration is optional, it's always available. The admin initialisation
	// output is used to configure and prepare other internal components for administration.
	zerologr.Info("Loading admin")
	admin := admin.New(
		&admin.Opts{
			Mux:    mux,
			DB:     db,
			OASDir: internalenv.OASDirectory.Value(),
			Cfg:    cfg.AdminConfig,
		},
	)

	zerologr.Info("Loading observability")
	observability := obs.NewComponent(&obs.Opts{
		Cfg:     cfg.ObservabilityConfig,
		Version: internalenv.Version.Value(),
	})

	zerologr.Info("Loading router")
	router := router.NewComponent(&router.Opts{Cfg: cfg.RouterConfig})

	zerologr.Info("Loading custom")
	customFlowComponents := make([]composer.FlowComponent, 0)

	if cfg.AuthEnabled() {
		zerologr.Info("Loading auth")
		authorizer := auth.NewComponent(&auth.Opts{
			Cfg:                    cfg.AuthConfig,
			Mux:                    mux,
			DB:                     db,
			OASDir:                 internalenv.OASDirectory.Value(),
			AdminSessionMiddleware: admin.SessionMiddleware,
		})
		customFlowComponents = append(customFlowComponents, authorizer)
	}

	if cfg.OASEnabled() {
		zerologr.Info("Loading OAS validator")
		oasValidator := oas.NewComponent(&oas.Opts{
			Cfg: cfg.OASConfig,
		})
		customFlowComponents = append(customFlowComponents, oasValidator)

		// Register the OAS validator with the admin component so that it can serve OAS to the admin API.
		admin.SetOASBackend(oasValidator)
	}

	custom := custom.NewComponent(customFlowComponents...)

	zerologr.Info("Loading forwarder")
	forwarder := forwarder.NewComponent(&forwarder.Opts{})

	zerologr.Info("Loading composer")
	composer := composer.New(&composer.Opts{
		Observability: observability,
		Router:        router,
		Custom:        custom,
		Forwarder:     forwarder,
	})

	// Register the flow fetcher with the admin component so that it can serve flow metadata to the admin API.
	admin.SetFlowFetcher(composer)

	zerologr.Info("Starting server")
	mux.Handle("/gw/", composer)
	server := http.Server{
		Addr:         fmt.Sprintf(":%d", internalenv.Port.Value()),
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
