package cmd

import (
	"errors"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// askYesNo shows a yes/no confirmation prompt with the given question and
// description and returns the user's answer. An empty description is not shown.
func askYesNo(question, description string) (bool, error) {
	var yes bool

	form := huh.NewForm(huh.NewGroup(newConfirm(question, description, &yes)))

	if err := form.Run(); err != nil {
		return false, err
	}

	return yes, nil
}

// newConfirm builds a yes/no field with left-aligned buttons. An empty description is
// not shown.
func newConfirm(title, description string, value *bool) *huh.Confirm {
	confirm := huh.NewConfirm().
		Title(title).
		WithButtonAlignment(lipgloss.Left).
		Value(value)

	if description != "" {
		confirm = confirm.Description(description)
	}

	return confirm
}

// newPostgresConfirm builds the question that chooses PostgreSQL over SQLite, shared by the
// compose and config commands.
func newPostgresConfirm(value *bool) *huh.Confirm {
	return newConfirm(
		"Include PostgreSQL as the persistence backend?",
		"Uses SQLite by default if skipped.",
		value,
	)
}

// runInteractive runs the interactive prompts in collect. The command usage is not printed
// for errors from an interactive session, and cancelling a prompt (for example with ctrl+c)
// exits without any output.
func runInteractive(cmd *cobra.Command, collect func() error) error {
	cmd.SilenceUsage = true

	err := collect()
	if errors.Is(err, huh.ErrUserAborted) {
		cmd.SilenceErrors = true
	}

	return err
}
