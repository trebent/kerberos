package env

import (
	"fmt"

	"github.com/trebent/envparser"
)

var (
	port = envparser.Register(&envparser.Opts[int]{
		Name:  "PORT",
		Desc:  "Port for the Kerberos API GW server.",
		Value: 30000,
		Validate: func(v int) error {
			if v < 1000 || v > 65535 {
				return fmt.Errorf("must be between 1000 and 65535: %d", v)
			}
			return nil
		},
	})
)
