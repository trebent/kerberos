package env

import (
	"fmt"

	"github.com/trebent/envparser"
)

var (
	Port = envparser.Register(&envparser.Opts[int]{
		Name:  "PORT",
		Desc:  "Port for the Kerberos API GW server.",
		Value: 30000, // nolint: mnd
		Validate: func(v int) error {
			if v < 1000 || v > 65535 {
				return fmt.Errorf("must be between 1000 and 65535: %d", v)
			}
			return nil
		},
	})
	ReadTimeoutSeconds = envparser.Register(&envparser.Opts[int]{
		Name:  "READ_TIMEOUT_SECONDS",
		Desc:  "Read timeout in seconds.",
		Value: 5, // nolint: mnd
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
		Value: 5, // nolint: mnd
		Validate: func(v int) error {
			if v < 1 {
				return fmt.Errorf("must be greater than 0: %d", v)
			}
			return nil
		},
	})
	TestEndpoint = envparser.Register(&envparser.Opts[bool]{
		Name:  "TEST_ENDPOINT",
		Desc:  "Set to enable a testing endpoint that can be used to test how Kerberos generates metrics for various requests.",
		Value: false,
	})
)
