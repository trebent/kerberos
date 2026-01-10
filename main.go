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
	"github.com/trebent/kerberos/internal/auth"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/custom"
	"github.com/trebent/kerberos/internal/composer/forwarder"
	obs "github.com/trebent/kerberos/internal/composer/observability"
	"github.com/trebent/kerberos/internal/composer/router"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/db/sqlite"
	internalenv "github.com/trebent/kerberos/internal/env"
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

	cleanup, err := obs.Instrument(signalCtx, cfg, serviceName, internalenv.Version.Value())
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
	zerologr.Info("Setting up configuration...")

	cfg := config.New()

	var (
		err              error
		obsConfigName    string
		routerConfigName string
		authConfigName   string
	)

	// Register all configurations.
	obsConfigName, err = obs.RegisterWith(cfg)
	must(err)
	routerConfigName, err = router.RegisterWith(cfg)
	must(err)
	authConfigName, err = auth.RegisterWith(cfg)
	must(err)

	// Load all input configuration data.
	if internalenv.ObsJSONFile.Value() != "" {
		zerologr.Info("Observability configuration detected, loading")
		obsData, _ := os.ReadFile(internalenv.ObsJSONFile.Value())
		cfg.MustLoad(obsConfigName, obsData)
	}

	if internalenv.AuthJSONFile.Value() != "" {
		zerologr.Info("Auth configuration detected, loading")
		authData, _ := os.ReadFile(internalenv.AuthJSONFile.Value())
		cfg.MustLoad(authConfigName, authData)
	}

	routerData, _ := os.ReadFile(internalenv.RouteJSONFile.Value())
	if len(routerData) == 0 {
		zerologr.Error(errors.New("missing router data"), "Router data empty")
		os.Exit(1)
	}
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
	db := sqlite.New(
		&sqlite.Opts{DSN: filepath.Join(internalenv.DBDirectory.Value(), sqlite.DBName)},
	)

	zerologr.Info("Loading observability")
	observability := obs.NewComponent(&obs.Opts{Cfg: cfg})
	zerologr.Info("Loading router")
	router := router.NewComponent(&router.Opts{Cfg: cfg})

	zerologr.Info("Loading custom")
	customFlowComponents := make([]composertypes.FlowComponent, 0)

	if internalenv.AuthJSONFile.Value() != "" {
		zerologr.Info("Loading auth")
		authorizer := auth.New(&auth.Opts{
			Cfg: cfg,
			Mux: mux,
			DB:  db,
		})
		customFlowComponents = append(customFlowComponents, authorizer)
	}
	custom := custom.NewComponent(customFlowComponents...)

	zerologr.Info("Loading forwarder")
	forwarder := forwarder.NewComponent(
		&forwarder.Opts{
			TargetContextKey: composertypes.TargetContextKey,
		},
	)

	composer := composer.New(&composer.Opts{
		Observability: observability,
		Router:        router,
		Custom:        custom,
		Forwarder:     forwarder,
	})

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

func must(err error) {
	if err != nil {
		panic(err)
	}
}
