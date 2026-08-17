// Package cmd contains all krbctl CLI commands.
package cmd

import (
	"github.com/spf13/cobra"
)

// Execute adds all child commands to the root command and runs it.
func Execute() error {
	root := &cobra.Command{
		Use:   "krbctl",
		Short: "krbctl is the Kerberos deployment CLI",
		Long: `krbctl helps you set up a Kerberos deployment by generating
a base compose.yaml and a base kerberos configuration file through
interactive prompts.`,
	}

	root.AddCommand(newComposeCmd())
	root.AddCommand(newConfigCmd())

	return root.Execute()
}
