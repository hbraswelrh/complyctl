// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/oscal-compass/oscal-sdk-go/transformers"
	"github.com/oscal-compass/oscal-sdk-go/validation"
	"github.com/spf13/cobra"

	"github.com/complytime/complytime/cmd/complytime/option"
	"github.com/complytime/complytime/internal/complytime"
)

const assessmentPlanLocation = "assessment-plan.json"

// PlanOptions defines options for the "plan" subcommand
type planOptions struct {
	*option.Common
	complyTimeOpts *option.ComplyTime

	// dryRun loads the defaults and prints the config to stdout
	dryRun bool
}

// planCmd creates a new cobra.Command for the "plan" subcommand
func planCmd(common *option.Common) *cobra.Command {
	planOpts := &planOptions{
		Common:         common,
		complyTimeOpts: &option.ComplyTime{},
	}
	cmd := &cobra.Command{
		Use:     "plan [flags] id",
		Short:   "Generate a new assessment plan for a given compliance framework id.",
		Example: "complytime plan myframework",
		Args:    cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				planOpts.complyTimeOpts.FrameworkID = filepath.Clean(args[0])
			}
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPlan(cmd, planOpts)
		},
	}
	cmd.Flags().BoolVarP(&planOpts.dryRun, "dry-run", "n", false, "load the defaults and print the config to stdout")
	planOpts.complyTimeOpts.BindFlags(cmd.Flags())
	return cmd
}

func runPlan(cmd *cobra.Command, opts *planOptions) error {
	// Create the application directory if it does not exist
	appDir, err := complytime.NewApplicationDirectory(true)
	if err != nil {
		return err
	}
	logger.Debug(fmt.Sprintf("Using application directory: %s", appDir.AppDir()))

	validator := validation.NewSchemaValidator()
	componentDefs, err := complytime.FindComponentDefinitions(appDir.BundleDir(), validator)
	if err != nil {
		return err
	}

	if opts.dryRun {
		// Write the plan configuration to stdout
		return planDryRun(opts.complyTimeOpts.FrameworkID, componentDefs)
	}

	logger.Debug(fmt.Sprintf("Using bundle directory: %s for component definitions.", appDir.BundleDir()))
	assessmentPlan, err := transformers.ComponentDefinitionsToAssessmentPlan(cmd.Context(), componentDefs, opts.complyTimeOpts.FrameworkID)
	if err != nil {
		return err
	}

	filePath := filepath.Join(opts.complyTimeOpts.UserWorkspace, assessmentPlanLocation)
	cleanedPath := filepath.Clean(filePath)

	if err := complytime.WritePlan(assessmentPlan, opts.complyTimeOpts.FrameworkID, cleanedPath); err != nil {
		return fmt.Errorf("error writing assessment plan to %s: %w", cleanedPath, err)
	}
	logger.Info(fmt.Sprintf("Assessment plan written to %s\n", cleanedPath))
	return nil
}

// loadPlan returns the loaded assessment plan and path from the workspace.
func loadPlan(opts *option.ComplyTime, validator validation.Validator) (*oscalTypes.AssessmentPlan, string, error) {
	apPath := filepath.Join(opts.UserWorkspace, assessmentPlanLocation)
	apCleanedPath := filepath.Clean(apPath)
	assessmentPlan, err := complytime.ReadPlan(apCleanedPath, validator)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", fmt.Errorf("error: assessment plan does not exist in workspace %s: %w\n\nDid you run the plan command?",
				opts.UserWorkspace,
				err)
		}
		return nil, "", err
	}
	return assessmentPlan, apCleanedPath, nil
}
