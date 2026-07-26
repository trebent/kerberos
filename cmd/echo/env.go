package main

import "github.com/trebent/envparser"

var (
	logToConsole = envparser.Register(&envparser.Opts[bool]{
		Name:  "LOG_TO_CONSOLE",
		Desc:  "Set to log to console.",
		Value: true,
	})
	verbosity = envparser.Register(&envparser.Opts[int]{
		Name:  "LOG_VERBOSITY",
		Desc:  "Sets the logging verbosity level.",
		Value: defaultLogVerbosity,
	})
	version = envparser.Register(&envparser.Opts[string]{
		Name:  "VERSION",
		Desc:  "Sets the application version.",
		Value: "unset",
	})
	port = envparser.Register(&envparser.Opts[int]{
		Name:  "PORT",
		Desc:  "Port to listen on.",
		Value: defaultPort,
	})
	observabilityEnabled = envparser.Register(&envparser.Opts[bool]{
		Name:  "OBSERVABILITY_ENABLED",
		Desc:  "Enables or disables observability features.",
		Value: true,
	})
	tlsCertFile = envparser.Register(&envparser.Opts[string]{
		Name: "TLS_CERT_FILE",
		Desc: "Path to the PEM-encoded server certificate file. When set together with TLS_KEY_FILE, the server enables TLS.",
	})
	tlsKeyFile = envparser.Register(&envparser.Opts[string]{
		Name: "TLS_KEY_FILE",
		Desc: "Path to the PEM-encoded server private key file. When set together with TLS_CERT_FILE, the server enables TLS.",
	})
	tlsClientCAFile = envparser.Register(&envparser.Opts[string]{
		Name: "TLS_CLIENT_CA_FILE",
		Desc: "Path to a PEM-encoded CA certificate bundle used to verify client certificates (mTLS). Requires TLS_CERT_FILE and TLS_KEY_FILE to be set.",
	})
)
