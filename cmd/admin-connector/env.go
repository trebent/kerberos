package main

import (
	"fmt"
	"strings"

	"github.com/trebent/envparser"
	"github.com/trebent/kerberos/internal/env"
)

const noWhitelist = ""

var (
	logToConsole = envparser.Register(&envparser.Opts[bool]{
		Name: "LOG_TO_CONSOLE",
		Desc: "Set to log to console.",
	})
	logVerbosity = envparser.Register(&envparser.Opts[int]{
		Name:     "LOG_VERBOSITY",
		Desc:     "Set the log verbosity.",
		Validate: env.ValidateGreaterThanOrEqualToZero,
	})
	version = envparser.Register(&envparser.Opts[string]{
		Name:  "VERSION",
		Desc:  "Sets the application version.",
		Value: "unset",
	})
	observabilityEnabled = envparser.Register(&envparser.Opts[bool]{
		Name:  "OBSERVABILITY_ENABLED",
		Desc:  "Enables or disables observability features.",
		Value: true,
	})
	runtimeMetrics = envparser.Register(&envparser.Opts[bool]{
		Name:  "RUNTIME_METRICS",
		Desc:  "Set to true to expose runtime metrics.",
		Value: true,
	})

	port = envparser.Register(&envparser.Opts[int]{
		Name:     "PORT",
		Desc:     "Port for the admin connector.",
		Value:    30100, // nolint: mnd
		Validate: env.ValidatePort,
	})
	target = envparser.Register(&envparser.Opts[string]{
		Name:     "TARGET",
		Desc:     "Target for the admin connector.",
		Required: true,
		Validate: env.ValidateHost,
	})

	readTimeoutSeconds = envparser.Register(&envparser.Opts[int]{
		Name:     "READ_TIMEOUT_SECONDS",
		Desc:     "Read timeout in seconds.",
		Value:    5, // nolint: mnd
		Validate: env.ValidateGreaterThanZero,
	})
	writeTimeoutSeconds = envparser.Register(&envparser.Opts[int]{
		Name:     "WRITE_TIMEOUT_SECONDS",
		Desc:     "Write timeout in seconds.",
		Value:    5, // nolint: mnd
		Validate: env.ValidateGreaterThanZero,
	})

	whitelist = envparser.Register(&envparser.Opts[string]{
		Name:  "WHITELIST",
		Desc:  "Comma-separated list of whitelisted Origins.",
		Value: noWhitelist,
		Validate: func(s string) error {
			if s == "" {
				return nil
			}

			split := strings.Split(s, ",")

			if len(split) == 0 {
				return nil
			}

			for i, origin := range split {
				if len(origin) == 0 {
					return fmt.Errorf("origin at index %d is empty", i)
				}
			}

			return nil
		},
	})

	serverCertFile = envparser.Register(&envparser.Opts[string]{
		Name:     "SERVER_CERT_FILE",
		Desc:     "Path to the server certificate file.",
		Value:    "",
		Validate: env.ValidateDirPath,
	})
	serverKeyFile = envparser.Register(&envparser.Opts[string]{
		Name:     "SERVER_KEY_FILE",
		Desc:     "Path to the server key file.",
		Value:    "",
		Validate: env.ValidateDirPath,
	})
)

func getWhitelist() []string {
	if whitelist.Value() == noWhitelist {
		return []string{}
	}

	return strings.Split(whitelist.Value(), ",")
}
