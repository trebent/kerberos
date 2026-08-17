package main

import (
	"os"

	"github.com/trebent/kerberos/cmd/krbctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
