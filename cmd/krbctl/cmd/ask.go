package cmd

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// askYesNo shows a yes/no confirmation prompt with the given question and
// description and returns the user's answer. An empty description is not shown.
func askYesNo(question, description string) (bool, error) {
	var yes bool

	confirm := huh.NewConfirm().
		Title(question).
		WithButtonAlignment(lipgloss.Left).
		Value(&yes)

	if description != "" {
		confirm = confirm.Description(description)
	}

	form := huh.NewForm(huh.NewGroup(confirm))

	if err := form.Run(); err != nil {
		return false, err
	}

	return yes, nil
}
