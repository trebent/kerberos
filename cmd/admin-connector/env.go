package main

import (
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
)
