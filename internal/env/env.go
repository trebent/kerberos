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
)

func Parse() error {
	return envparser.Parse()
}
