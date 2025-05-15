// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/oscal-compass/oscal-sdk-go/transformers"
	"github.com/oscal-compass/oscal-sdk-go/validation"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/complytime/complytime/cmd/complytime/option"
	"github.com/complytime/complytime/internal/complytime"
)

const assessmentPlanLocation = "assessment-plan.json"
const assessmentPlanFilterLocation = "assessment-plan-filter.yml"

// PlanOptions defines options for the "plan" subcommand
type planOptions struct {
	*option.Common
	complyTimeOpts *option.ComplyTime

	// dryRun loads the defaults and prints the config to stdout
	dryRun bool

	// loadConfig reads "config.yml" to pre-tailor the generated assessment plan
	loadConfig bool
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
	cmd.Flags().BoolVarP(&planOpts.loadConfig, "load-config", "l", false, "load assessment-plan-filter.yml to pre-tailor the generated assessment plan")
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

	if opts.loadConfig {
		// Read assessment plan filter
		apfBytes, err := os.ReadFile(filepath.Join(opts.complyTimeOpts.UserWorkspace, assessmentPlanFilterLocation))
		if err != nil {
			return fmt.Errorf("error reading assessment plan filter: %w", err)
		}
		apf := PlanData{}
		if err := yaml.Unmarshal(apfBytes, &apf); err != nil {
			return fmt.Errorf("error unmarshaling assessment plan filter: %w", err)
		}
		if err := filterAssessmentPlan(cmd.Context(), assessmentPlan, apf); err != nil {
			return fmt.Errorf("error filtering assessment plan: %w", err)
		}
	}

	filePath := filepath.Join(opts.complyTimeOpts.UserWorkspace, assessmentPlanLocation)
	cleanedPath := filepath.Clean(filePath)

	if err := complytime.WritePlan(assessmentPlan, opts.complyTimeOpts.FrameworkID, cleanedPath); err != nil {
		return fmt.Errorf("error writing assessment plan to %s: %w", cleanedPath, err)
	}
	logger.Info(fmt.Sprintf("Assessment plan written to %s\n", cleanedPath))
	return nil
}

func filterAssessmentPlan(ctx context.Context, assessmentPlan *oscalTypes.AssessmentPlan, assessmentPlanFilter PlanData) error {

	//    /assessment-plan/local-definitions/activities/steps/reviewed-controls/control-selections/include-controls/control-id - Control Identifier Reference
	//    /assessment-plan/local-definitions/activities/steps/reviewed-controls/control-selections/exclude-controls/control-id - Control Identifier Reference
	//    /assessment-plan/local-definitions/activities/related-controls/control-selections/include-controls/control-id - Control Identifier Reference
	//    /assessment-plan/local-definitions/activities/related-controls/control-selections/exclude-controls/control-id - Control Identifier Reference
	//    /assessment-plan/reviewed-controls/control-selections/include-controls/control-id - Control Identifier Reference
	//    /assessment-plan/reviewed-controls/control-selections/exclude-controls/control-id - Control Identifier Reference

	//    /assessment-plan/local-definitions/activities/steps/reviewed-controls/control-selections/include-controls - Select Control
	//    /assessment-plan/local-definitions/activities/related-controls/control-selections/include-controls - Select Control
	//    /assessment-plan/reviewed-controls/control-selections/include-controls - Select Control

	//    /assessment-plan/local-definitions/activities/steps/reviewed-controls/control-selections/include-controls/statement-ids - Include Specific Statements
	//    /assessment-plan/local-definitions/activities/steps/reviewed-controls/control-selections/exclude-controls/statement-ids - Include Specific Statements
	//    /assessment-plan/local-definitions/activities/related-controls/control-selections/include-controls/statement-ids - Include Specific Statements
	//    /assessment-plan/local-definitions/activities/related-controls/control-selections/exclude-controls/statement-ids - Include Specific Statements
	//    /assessment-plan/reviewed-controls/control-selections/include-controls/statement-ids - Include Specific Statements
	//    /assessment-plan/reviewed-controls/control-selections/exclude-controls/statement-ids - Include Specific Statements

	// "Any control specified within exclude-controls must first be within a range of explicitly included controls, via include-controls or include-all."

	includedControls := map[string]bool{}
	for _, id := range assessmentPlanFilter.Controls {
		includedControls[id] = true
	}

	if assessmentPlan.LocalDefinitions != nil {
		if assessmentPlan.LocalDefinitions.Activities != nil {
			for activityI := range *assessmentPlan.LocalDefinitions.Activities {
				var activity *oscalTypes.Activity
				activity = &(*assessmentPlan.LocalDefinitions.Activities)[activityI]
				if activity.RelatedControls != nil && activity.RelatedControls.ControlSelections != nil {
					for controlSelectionI := range activity.RelatedControls.ControlSelections {
						var controlSelection *oscalTypes.AssessedControls
						controlSelection = &activity.RelatedControls.ControlSelections[controlSelectionI]
						filterControlSelection(controlSelection, includedControls)
					}
				}

				if activity.Steps != nil {
					for stepI := range *activity.Steps {
						var step *oscalTypes.Step
						step = &(*activity.Steps)[stepI]
						if step.ReviewedControls == nil {
							continue
						}
						if step.ReviewedControls.ControlSelections == nil {
							continue
						}
						for controlSelectionI := range step.ReviewedControls.ControlSelections {
							var controlSelection *oscalTypes.AssessedControls
							controlSelection = &step.ReviewedControls.ControlSelections[controlSelectionI]
							filterControlSelection(controlSelection, includedControls)
						}
					}
				}
			}
		}
	}

	if assessmentPlan.ReviewedControls.ControlSelections != nil {
		for controlSelectionI := range assessmentPlan.ReviewedControls.ControlSelections {
			var controlSelection *oscalTypes.AssessedControls
			controlSelection = &assessmentPlan.ReviewedControls.ControlSelections[controlSelectionI]
			filterControlSelection(controlSelection, includedControls)
		}
	}

	return nil
}

// filterControlSelection makes inclusions explicit
func filterControlSelection(controlSelection *oscalTypes.AssessedControls, includedControls map[string]bool) {
	// The new included controls should be the intersection of
	// the originally included controls and the newly included controls.
	// ExcludedControls are preserved.

	// includedControls specifies everything we allow - do not include all
	includedAll := controlSelection.IncludeAll != nil
	controlSelection.IncludeAll = nil

	originalIncludedControls := map[string]bool{}
	if controlSelection.IncludeControls != nil {
		for _, controlId := range *controlSelection.IncludeControls {
			originalIncludedControls[controlId.ControlId] = true
		}
	}
	var newIncludedControls []oscalTypes.AssessedControlsSelectControlById
	for controlId := range includedControls {
		if includedAll || originalIncludedControls[controlId] {
			newIncludedControls = append(newIncludedControls, oscalTypes.AssessedControlsSelectControlById{
				ControlId: controlId,
			})
		}
	}
	controlSelection.IncludeControls = &newIncludedControls
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
