// Package cmd contains all krbctl CLI commands.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is the krbctl version. It defaults to "unset" and is overridden at
// build time via -ldflags "-X github.com/trebent/kerberos/cmd/krbctl/cmd.version=...".
var version = "unset"

// Execute adds all child commands to the root command and runs it.
func Execute() error {
	root := &cobra.Command{
		Use:     "krbctl",
		Version: version,
		Short:   "krbctl is the Kerberos deployment CLI",
		Long: fmt.Sprintf(`krbctl helps you set up a Kerberos deployment by generating
a base compose.yaml and a base kerberos configuration file through
interactive prompts.

Version: %s`, version),
	}

	root.AddCommand(newComposeCmd())
	root.AddCommand(newConfigCmd())

	return root.Execute()
}
