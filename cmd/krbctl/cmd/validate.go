package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/trebent/kerberos/internal/config"
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "validate <file>",
		Aliases: []string{"v"},
		Short:   "Validate a Kerberos (or, with -c, admin-connector) config file",
		Args:    cobra.ExactArgs(1),
		// A failed validation is not a usage error.
		SilenceUsage: true,
		RunE:         runValidate,
	}

	cmd.Flags().BoolP("connector", "c", false, "Validate as an admin-connector config.")

	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	connector, err := cmd.Flags().GetBool("connector")
	if err != nil {
		return err
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}

	if connector {
		_, err = config.ParseConnector(data)
	} else {
		_, err = config.ParseKerberos(data)
	}

	if err != nil {
		return fmt.Errorf("%s: %w", args[0], err)
	}

	fmt.Fprintf(os.Stdout, "%s: ok\n", args[0])

	return nil
}
