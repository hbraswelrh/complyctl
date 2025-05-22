// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/goccy/go-yaml"
	"github.com/oscal-compass/oscal-sdk-go/transformers"
	"github.com/oscal-compass/oscal-sdk-go/validation"
	"github.com/spf13/cobra"

	"github.com/complytime/complytime/cmd/complytime/option"
	"github.com/complytime/complytime/internal/complytime"
	"github.com/complytime/complytime/internal/complytime/plan"
)

const assessmentPlanLocation = "assessment-plan.json"

// PlanOptions defines options for the "plan" subcommand
type planOptions struct {
	*option.Common
	complyTimeOpts *option.ComplyTime

	// dryRun loads the defaults and prints the config to stdout
	dryRun bool

	// WithConfig "config.yml" to customize the generated assessment plan
	withConfig string
}

var planExample = `
	# The default behavior is to prepare a default assessment plan with all defined
    # controls within the framework in scope
	complytime plan myframework

	# To customize the assessment plan, run in dry-run mode
	complytime plan myframework --dry-run > config.yml

	# Alter the configuration and use it as input for plan customization
	complytime plan myframework --with-config config.yml
`

// planCmd creates a new cobra.Command for the "plan" subcommand
func planCmd(common *option.Common) *cobra.Command {
	planOpts := &planOptions{
		Common:         common,
		complyTimeOpts: &option.ComplyTime{},
	}
	cmd := &cobra.Command{
		Use:     "plan [flags] id",
		Short:   "Generate a new assessment plan for a given compliance framework id.",
		Example: planExample,
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
	cmd.Flags().BoolVarP(&planOpts.dryRun, "dry-run", "d", false, "load the defaults and print the config to stdout")
	cmd.Flags().StringVarP(&planOpts.withConfig, "with-config", "c", "", "load config.yml to customize the generated assessment plan")
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

	if opts.withConfig != "" {
		// Read assessment plan filter
		// FIXME: Is `assessment filter plan` the right location?
		// Seems more intuitive to write the plan content to a well-known location and load only
		// if present or allow the user to pass in the path. We could use a mutli-writer to write to the path and
		// stdout if desired.

		// TODO: HB - updated location for reading file - may need change for variable name
		configBytes, err := os.ReadFile(filepath.Join(opts.withConfig))
		if err != nil {
			return fmt.Errorf("error reading assessment plan: %w", err)
		}
		assessmentScope := plan.AssessmentScope{}
		if err := yaml.Unmarshal(configBytes, &assessmentScope); err != nil {
			return fmt.Errorf("error unmarshaling assessment plan: %w", err)
		}
		assessmentScope.Logger = logger
		assessmentScope.ApplyScope(assessmentPlan)
	}

	filePath := filepath.Join(opts.complyTimeOpts.UserWorkspace, assessmentPlanLocation)
	cleanedPath := filepath.Clean(filePath)

	if err := plan.WritePlan(assessmentPlan, opts.complyTimeOpts.FrameworkID, cleanedPath); err != nil {
		return fmt.Errorf("error writing assessment plan to %s: %w", cleanedPath, err)
	}
	logger.Info(fmt.Sprintf("Assessment plan written to %s\n", cleanedPath))
	return nil
}

// loadPlan returns the loaded assessment plan and path from the workspace.
func loadPlan(opts *option.ComplyTime, validator validation.Validator) (*oscalTypes.AssessmentPlan, string, error) {
	apPath := filepath.Join(opts.UserWorkspace, assessmentPlanLocation)
	apCleanedPath := filepath.Clean(apPath)
	assessmentPlan, err := plan.ReadPlan(apCleanedPath, validator)
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

// planDryRun leverages the AssessmentScope structure to populate tailoring config.
// The config is written to stdout.
func planDryRun(frameworkId string, cds []oscalTypes.ComponentDefinition) error {
	baseScope := plan.NewAssessmentScope(frameworkId, nil)
	if cds == nil {
		return fmt.Errorf("no component definitions found")
	}
	for _, componentDef := range cds {
		if componentDef.Components == nil {
			continue
		}
		for _, component := range *componentDef.Components {
			if component.ControlImplementations == nil {
				continue
			}
			// FIXME: Filter the added controls by the framework ID property on the
			// control implementation. This ensure only the applicable controls end up
			// in the configuration for review.
			// FIXME: logger statements should not include the filter location comment
			for _, ci := range *component.ControlImplementations {
				if ci.ImplementedRequirements == nil {
					continue
				}
				if ci.Props != nil {
					for _, frameworkVal := range *ci.Props {
						if baseScope.FrameworkID == frameworkVal.Value {
							continue
						}
						for _, ir := range ci.ImplementedRequirements {
							if ir.ControlId != "" {
								baseScope.IncludeControls = append(baseScope.IncludeControls, ir.ControlId)
							}
						}
					}
					for _, ir := range ci.ImplementedRequirements {
						if ir.ControlId != "" {
							baseScope.IncludeControls = append(baseScope.IncludeControls, ir.ControlId)
						}
					}
				}
			}
		}
	}

	out, err := yaml.Marshal(&baseScope)
	if err != nil {
		return fmt.Errorf("error marshalling yaml content: %v", err)
	}
	fmt.Println(string(out))
	return nil
}
