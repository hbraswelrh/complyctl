// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/oscal-compass/compliance-to-policy-go/v2/framework"
	"github.com/oscal-compass/oscal-sdk-go/extensions"
	"github.com/oscal-compass/oscal-sdk-go/validation"
	"github.com/spf13/cobra"

	"github.com/complytime/complytime/cmd/complytime/option"
	"github.com/complytime/complytime/internal/complytime"
)

// generateOptions defines options for the "generate" subcommand
type generateOptions struct {
	*option.Common
	complyTimeOpts *option.ComplyTime
}

// generateCmd creates a new cobra.Command for the "generate" subcommand
func generateCmd(common *option.Common) *cobra.Command {
	generateOpts := &generateOptions{
		Common:         common,
		complyTimeOpts: &option.ComplyTime{},
	}
	cmd := &cobra.Command{
		Use:     "generate [flags]",
		Short:   "Generate PVP policy from an assessment plan",
		Example: "complytime generate",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGenerate(cmd, generateOpts)
		},
	}
	generateOpts.complyTimeOpts.BindFlags(cmd.Flags())
	return cmd
}

func runGenerate(cmd *cobra.Command, opts *generateOptions) error {
	validator := validation.NewSchemaValidator()
	ap, _, err := loadPlan(opts.complyTimeOpts, validator)
	if err != nil {
		return err
	}

	planSettings, err := getPlanSettings(opts.complyTimeOpts, ap)
	if err != nil {
		return err
	}

	// Set the framework ID from state (assessment plan)
	frameworkProp, valid := extensions.GetTrestleProp(extensions.FrameworkProp, *ap.Metadata.Props)
	if !valid {
		return fmt.Errorf("error reading framework property from assessment plan")
	}
	opts.complyTimeOpts.FrameworkID = frameworkProp.Value

	// Create the application directory if it does not exist
	appDir, err := complytime.NewApplicationDirectory(true)
	if err != nil {
		return err
	}
	logger.Debug(fmt.Sprintf("Using application directory: %s", appDir.AppDir()))
	cfg, err := complytime.Config(appDir, validator)
	if err != nil {
		return err
	}
	logger.Debug("The configuration from the C2PConfig was successfully loaded.")

	// set config logger to CLI charm logger
	cfg.Logger = logger

	manager, err := framework.NewPluginManager(cfg)
	if err != nil {
		return fmt.Errorf("error initializing plugin manager: %w", err)
	}

	pluginOptions := opts.complyTimeOpts.ToPluginOptions()
	plugins, cleanup, err := complytime.Plugins(manager, pluginOptions)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return fmt.Errorf("errors launching plugins: %w", err)
	}

	err = manager.GeneratePolicy(cmd.Context(), plugins, planSettings)
	if err != nil {
		return err
	}
	logger.Info("Policy generation process completed for available plugins.")
	return nil
}
