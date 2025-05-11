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

	"github.com/trebent/envparser"
	"github.com/trebent/kerberos/internal/env"
	"github.com/trebent/zerologr"
)

var (
	readTimeout  time.Duration
	writeTimeout time.Duration
)

//nolint:gochecknoinits
func init() {
	envparser.Prefix = "KRB" //nolint:reassign
}

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

	readTimeout = time.Duration(env.ReadTimeoutSeconds.Value()) * time.Second
	writeTimeout = time.Duration(env.WriteTimeoutSeconds.Value()) * time.Second

	// Set up monitoring
	logger := zerologr.New(&zerologr.Opts{
		Console: env.LogToConsole.Value(),
		Caller:  true,
		V:       env.LogVerbosity.Value(),
	})

	// Start Kerberos API GW server
	if err := startServer(); !errors.Is(err, http.ErrServerClosed) {
		logger.Error(err, "Failed to start Kerberos HTTP server")
		os.Exit(1)
	}
}

func startServer() error {
	handler := http.NewServeMux()

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

	signalCtx, signalCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer signalCancel()
	<-signalCtx.Done()

	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), readTimeout+writeTimeout)
	defer timeoutCancel()

	return errors.Join(server.Shutdown(timeoutCtx), <-errChan)
}
