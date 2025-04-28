// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/oscal-compass/oscal-sdk-go/validation"
	"github.com/spf13/cobra"

	"github.com/complytime/complytime/cmd/complytime/option"
	"github.com/complytime/complytime/internal/complytime"
	"github.com/complytime/complytime/internal/terminal"
)

// listOptions defines options for the "list" subcommand
type listOptions struct {
	*option.Common
	// print a plain table only
	plain bool
}

// listCmd creates a new cobra.Command for the "list" subcommand
func listCmd(common *option.Common) *cobra.Command {
	listOpts := &listOptions{
		Common: common,
	}
	cmd := &cobra.Command{
		Use:          "list [flags]",
		Short:        "List information about supported frameworks and components.",
		SilenceUsage: true,
		Example:      "complytime list",
		Args:         cobra.NoArgs,
		RunE:         func(_ *cobra.Command, _ []string) error { return runList(listOpts) },
	}
	cmd.Flags().BoolVarP(&listOpts.plain, "plain", "p", false, "print the table with minimal formatting")
	return cmd
}

func runList(opts *listOptions) error {
	appDir, err := complytime.NewApplicationDirectory(true)
	if err != nil {
		return err
	}
	logger.Debug(fmt.Sprintf("Using application directory: %s", appDir.AppDir()))

	validator := validation.NewSchemaValidator()
	frameworks, err := complytime.LoadFrameworks(appDir, validator)
	if err != nil {
		return err
	}

	if opts.plain {
		terminal.ShowDefinitionTable(opts.Out, frameworks)
	} else {
		model := terminal.ShowPrettyDefinitionTable(frameworks)
		if _, err := tea.NewProgram(model, tea.WithOutput(opts.Out)).Run(); err != nil {
			return fmt.Errorf("failed to display component list: %w", err)
		}
	}
	return nil
}
