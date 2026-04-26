//nolint:gochecknoglobals // Package env provides environment variable parsing for the Kerberos service.
package env

import (
	"github.com/trebent/envparser"
)

var (
	Version = envparser.Register(&envparser.Opts[string]{
		Name:  "VERSION",
		Desc:  "Version of the Kerberos service.",
		Value: "unset",
	})

	LogToConsole = envparser.Register(&envparser.Opts[bool]{
		Name: "LOG_TO_CONSOLE",
		Desc: "Set to log to console.",
	})
	LogVerbosity = envparser.Register(&envparser.Opts[int]{
		Name:     "LOG_VERBOSITY",
		Desc:     "Set the log verbosity.",
		Validate: validateGreaterThanOrEqualToZero,
	})

	Port = envparser.Register(&envparser.Opts[int]{
		Name:     "PORT",
		Desc:     "Port for the Kerberos API GW server.",
		Value:    30000, // nolint: mnd
		Validate: validatePort,
	})
	AdminPort = envparser.Register(&envparser.Opts[int]{
		Name:     "ADMIN_PORT",
		Desc:     "Port for the Kerberos admin server.",
		Value:    30001, // nolint: mnd
		Validate: validatePort,
	})
	ReadTimeoutSeconds = envparser.Register(&envparser.Opts[int]{
		Name:     "READ_TIMEOUT_SECONDS",
		Desc:     "Read timeout in seconds.",
		Value:    5, // nolint: mnd
		Validate: validateGreaterThanZero,
	})
	WriteTimeoutSeconds = envparser.Register(&envparser.Opts[int]{
		Name:     "WRITE_TIMEOUT_SECONDS",
		Desc:     "Write timeout in seconds.",
		Value:    5, // nolint: mnd
		Validate: validateGreaterThanZero,
	})

	OASDirectory = envparser.Register(&envparser.Opts[string]{
		Name:     "OAS_DIRECTORY",
		Desc:     "Path to the directory where Kerberos OAS specifications are stored.",
		Value:    "",
		Validate: validateDirPath,
	})
)

func Parse() error {
	return envparser.Parse()
}
