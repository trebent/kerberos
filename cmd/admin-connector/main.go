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

	intotel "github.com/trebent/kerberos/internal/otel"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/kerberos/internal/security"

	"github.com/trebent/envparser"
	"github.com/trebent/zerologr"
)

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
	_ = envparser.Parse()

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

	if err := startServer(signalCtx); err != nil {
		zerologr.Error(err, "Failed to start server")

		//nolint:gocritic // Exit with code 1 on error.
		os.Exit(1)
	}
}

func startServer(signalCtx context.Context) error {
	mux := http.NewServeMux()

	var (
		handler           = newHandler(opts{target: target.Value(), scheme: getScheme()})
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
		whitelist = getWhitelist()
	)

	if len(whitelist) == 0 {
		zerologr.Info("No CORS whitelist provided, allowing all origins")
		finalHandler = loggingMiddleware(security.CORSMiddleware(handler))
	} else {
		zerologr.Info(
			"CORS whitelist provided, allowing only specified origins",
			"whitelist", whitelist,
		)
		finalHandler = loggingMiddleware(security.WhitelistCORSMiddleware(whitelist, handler))
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
		if serverCertFile.Value() != "" && serverKeyFile.Value() != "" {
			zerologr.Info(
				"Starting TLS server",
				"certFile", serverCertFile.Value(), "keyFile", serverKeyFile.Value(),
			)
			errChan <- server.ListenAndServeTLS(serverCertFile.Value(), serverKeyFile.Value())
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

func getScheme() string {
	if serverCertFile.Value() != "" && serverKeyFile.Value() != "" {
		return "https"
	}
	return "http"
}
