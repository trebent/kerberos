package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/trebent/kerberos/internal/config"
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "validate",
		Aliases: []string{"val"},
		Short:   "Validate a Kerberos or admin-connector config file",
		// Without a subcommand, fail and print the usage so the subcommands are listed.
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return errors.New("choose what to validate: kerberos or connector")
		},
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:     "kerberos <file>",
			Aliases: []string{"krb"},
			Short:   "Validate a Kerberos config file",
			Args:    cobra.ExactArgs(1),
			// A failed validation is not a usage error.
			SilenceUsage: true,
			RunE:         validateFile(config.ParseKerberos),
		},
		&cobra.Command{
			Use:          "connector <file>",
			Aliases:      []string{"conn"},
			Short:        "Validate an admin-connector config file",
			Args:         cobra.ExactArgs(1),
			SilenceUsage: true,
			RunE:         validateFile(config.ParseConnector),
		},
	)

	return cmd
}

// validateFile returns a command action that reads the file named by the first argument and
// checks it with parse.
func validateFile[T any](parse func([]byte) (T, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		path := args[0]

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if _, err := parse(data); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s: ok\n", path)

		return nil
	}
}
