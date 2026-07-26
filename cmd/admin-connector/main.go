package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/db/postgres"
	"github.com/trebent/kerberos/internal/db/sqlite"
	intotel "github.com/trebent/kerberos/internal/otel"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/kerberos/internal/security"

	"github.com/trebent/envparser"
	"github.com/trebent/zerologr"
)

var configPath string

func main() {
	flag.StringVar(
		&configPath, "config", "", "Path to the connector configuration file (required).",
	)
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
	_ = envparser.Parse()

	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	logger := zerologr.New(&zerologr.Opts{
		Console: logToConsole.Value(),
		Caller:  true,
		V:       logVerbosity.Value(),
	})

	zerologr.Set(logger)

	zerologr.Info("Starting admin connector", "port", port.Value())

	signalCtx, signalCancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer signalCancel()

	if observabilityEnabled.Value() {
		zerologr.Info("Initializing OpenTelemetry instrumentation")
		shutdown, err := intotel.Instrument(
			signalCtx, "admin-connector", version.Value(), runtimeMetrics.Value(),
		)
		if err != nil {
			zerologr.Error(err, "Failed to initialize OpenTelemetry")

			//nolint:gocritic // Exit with code 1 on error.
			os.Exit(1)
		}

		//nolint:errcheck // best-effort shutdown, no need to handle error here
		defer shutdown(context.Background())
	}

	if err := startServer(signalCtx, cfg); err != nil {
		zerologr.Error(err, "Failed to start server")

		//nolint:gocritic // Exit with code 1 on error.
		os.Exit(1)
	}
}

func loadConfig(path string) (*config.ConnectorConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.ConnectorConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func startServer(signalCtx context.Context, cfg *config.ConnectorConfig) error {
	mux := http.NewServeMux()
	isTLS := isTLSEnabled(cfg)

	sqlClient, err := createSQLClient(cfg.Persistence)
	if err != nil {
		return fmt.Errorf("failed to create SQL client: %w", err)
	}

	handler, err := newHandler(opts{
		version:   version.Value(),
		target:    target.Value(),
		scheme:    getScheme(isTLS),
		sqlClient: sqlClient,
	})
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	var (
		finalHandler      http.Handler
		loggingMiddleware = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				//nolint:errcheck // guaranteed
				wrapper := response.NewResponseWrapper(w).(*response.Wrapper)
				next.ServeHTTP(wrapper, r)
				zerologr.Info(
					fmt.Sprintf("%s %s %d", r.Method, r.URL.Path, wrapper.StatusCode()),
				)
			})
		}
	)

	if len(cfg.Whitelist) == 0 {
		zerologr.Info("No CORS whitelist provided, allowing all origins")
		finalHandler = loggingMiddleware(security.CORSMiddleware(handler))
	} else {
		zerologr.Info(
			"CORS whitelist provided, allowing only specified origins",
			"whitelist", cfg.Whitelist,
		)
		finalHandler = loggingMiddleware(security.WhitelistCORSMiddleware(cfg.Whitelist, handler))
	}

	mux.Handle("/", finalHandler)

	server := &http.Server{
		Handler:      mux,
		Addr:         fmt.Sprintf(":%d", port.Value()),
		ReadTimeout:  time.Duration(readTimeoutSeconds.Value()) * time.Second,
		WriteTimeout: time.Duration(writeTimeoutSeconds.Value()) * time.Second,
	}

	errChan := make(chan error, 1)
	go func() {
		if isTLS {
			zerologr.Info(
				"Starting TLS server",
				"certFile", cfg.ServerTLS.CertFile, "keyFile", cfg.ServerTLS.KeyFile,
			)
			errChan <- server.ListenAndServeTLS(cfg.ServerTLS.CertFile, cfg.ServerTLS.KeyFile)
		} else {
			zerologr.Info("Starting server")
			errChan <- server.ListenAndServe()
		}
	}()

	select {
	case <-signalCtx.Done():
		zerologr.Info("Shutting down server")
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			time.Duration(readTimeoutSeconds.Value()+writeTimeoutSeconds.Value())*time.Second,
		)
		defer shutdownCancel()

		return server.Shutdown(shutdownCtx)
	case err := <-errChan:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			zerologr.Error(err, "Failed to start server")
			return err
		}
	}

	return nil
}

func isTLSEnabled(cfg *config.ConnectorConfig) bool {
	if cfg.ServerTLS == nil {
		return false
	}

	return cfg.ServerTLS.CertFile != "" && cfg.ServerTLS.KeyFile != ""
}

func getScheme(isTLS bool) string {
	if isTLS {
		return "https"
	}

	return "http"
}

func createSQLClient(cfg *config.PersistenceConfig) (db.SQLClient, error) {
	if cfg == nil {
		return nil, errors.New("persistence config is nil")
	}

	switch cfg.Driver {
	case "sqlite":
		zerologr.Info("Using SQLite persistence")

		return sqlite.New(&sqlite.Opts{
			DSN: cfg.Address,
		}), nil
	case "postgres":
		zerologr.Info("Using PostgreSQL persistence")

		hostPort := strings.Split(cfg.Address, ":")
		if len(hostPort) != 2 {
			panic("Invalid database address format. Expected host:port for PostgreSQL.")
		}
		host := hostPort[0]
		port, err := strconv.Atoi(hostPort[1])
		if err != nil {
			panic("Invalid port in database address: " + err.Error())
		}

		// Build postgres DSN from structured fields.
		dsn := "host=%s port=%d dbname=%s"
		params := []any{host, port, cfg.Database}

		if cfg.SSLMode != nil {
			dsn += " sslmode=%s"
			params = append(params, *cfg.SSLMode)
		}

		if cfg.Postgres.Username != nil && cfg.Postgres.Password != nil {
			dsn += " user=%s"
			params = append(params, *cfg.Postgres.Username)

			dsn += " password=%s"
			params = append(params, *cfg.Postgres.Password)
		}

		return postgres.New(&postgres.Opts{DSN: fmt.Sprintf(dsn, params...)}), nil
	default:
		return nil, fmt.Errorf("unsupported persistence type: %s", cfg.Driver)
	}
}
