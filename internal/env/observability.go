//nolint:gochecknoglobals // Package env provides environment variable parsing for the Kerberos service.
package env

import (
	"fmt"

	"github.com/trebent/envparser"
)

var (
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
)
