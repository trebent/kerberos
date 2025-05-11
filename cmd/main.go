package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/trebent/envparser"
	_ "github.com/trebent/kerberos/internal/env"
)

func init() {
	envparser.Prefix = "KRB"
}

func main() {
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.CommandLine.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\n")
		fmt.Fprintf(flag.CommandLine.Output(), envparser.Help())
	}

	flag.Parse()
	// ExitOnError = true
	_ = envparser.Parse()

	// Set up monitoring
}
