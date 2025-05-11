package env

import (
	"fmt"

	"github.com/trebent/envparser"
)

var (
	Port = envparser.Register(&envparser.Opts[int]{
		Name:  "PORT",
		Desc:  "Port for the Kerberos API GW server.",
		Value: 30000, //nolint:mnd
		Validate: func(v int) error {
			if v < 1000 || v > 65535 {
				return fmt.Errorf("must be between 1000 and 65535: %d", v)
			}
			return nil
		},
	})
	LogToConsole = envparser.Register(&envparser.Opts[bool]{
		Name: "LOG_TO_CONSOLE",
		Desc: "Set to log to console.",
	})
	LogVerbosity = envparser.Register(&envparser.Opts[int]{
		Name: "LOG_VERBOSITY",
		Desc: "Set the log verbosity.",
		Validate: func(v int) error {
			if v < 0 {
				return fmt.Errorf("must be greater than or equal to 0: %d", v)
			}
			return nil
		},
	})
	ReadTimeoutSeconds = envparser.Register(&envparser.Opts[int]{
		Name:  "READ_TIMEOUT_SECONDS",
		Desc:  "Read timeout in seconds.",
		Value: 5,
		Validate: func(v int) error {
			if v < 1 {
				return fmt.Errorf("must be greater than 0: %d", v)
			}
			return nil
		},
	})
	WriteTimeoutSeconds = envparser.Register(&envparser.Opts[int]{
		Name:  "WRITE_TIMEOUT_SECONDS",
		Desc:  "Write timeout in seconds.",
		Value: 5,
		Validate: func(v int) error {
			if v < 1 {
				return fmt.Errorf("must be greater than 0: %d", v)
			}
			return nil
		},
	})
)
