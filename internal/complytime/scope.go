// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"fmt"
	"sort"

	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/hashicorp/go-hclog"
	"github.com/oscal-compass/oscal-sdk-go/extensions"
	"github.com/oscal-compass/oscal-sdk-go/validation"
)

// ParameterEntry represents a parameter with its name and value
type ParameterEntry struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// ControlEntry represents a control in the assessment scope
type ControlEntry struct {
	ControlID        string           `yaml:"controlId"`
	ControlTitle     string           `yaml:"controlTitle"`
	IncludeRules     []string         `yaml:"includeRules"`
	ExcludeRules     []string         `yaml:"excludeRules,omitempty"`
	SelectParameters []ParameterEntry `yaml:"selectParameters,omitempty"`
}

// AssessmentScope sets up the yaml mapping type for writing to config file.
// Formats testdata as go struct.
type AssessmentScope struct {
	// FrameworkID is the identifier for the control set
	// in the Assessment Plan.
	FrameworkID string `yaml:"frameworkId"`
	// IncludeControls defines controls that are in scope
	// of an assessment.
	IncludeControls    []ControlEntry `yaml:"includeControls"`
	GlobalExcludeRules []string       `yaml:"globalExcludeRules,omitempty"`
}

// NewAssessmentScope creates an AssessmentScope struct for a given framework id.
func NewAssessmentScope(frameworkID string) AssessmentScope {
	return AssessmentScope{
		FrameworkID: frameworkID,
	}
}

// NewAssessmentScopeFromCDs creates and populates an AssessmentScope struct for a given framework id and set of
// OSCAL Component Definitions.
func NewAssessmentScopeFromCDs(frameworkId string, appDir ApplicationDirectory, validator validation.Validator, cds ...oscalTypes.ComponentDefinition) (AssessmentScope, error) {
	includeControls := make(includeControlsSet)
	controlTitles := make(map[string]string)
	setParameters := make(map[string][]string)
	scope := NewAssessmentScope(frameworkId)

	if cds == nil {
		return AssessmentScope{}, fmt.Errorf("no component definitions found")
	}

	// Map to store control titles by source to avoid loading the same source multiple times
	controlTitlesBySource := make(map[string]map[string]string)

	for _, componentDef := range cds {
		if componentDef.Components == nil {
			continue
		}
		for _, component := range *componentDef.Components {
			if component.ControlImplementations == nil {
				continue
			}
			for _, ci := range *component.ControlImplementations {
				if ci.ImplementedRequirements == nil {
					continue
				}
				if ci.Props != nil {
					frameworkProp, found := extensions.GetTrestleProp(extensions.FrameworkProp, *ci.Props)
					if !found || frameworkProp.Value != scope.FrameworkID {
						continue
					}

					// Checking once for control titles from source on the control implementation
					if validator != nil {
						// Check if source was already loaded
						if _, sourceLoaded := controlTitlesBySource[ci.Source]; !sourceLoaded {
							// Load all titles from this source
							loadedTitles, err := loadControlTitlesFromSource(ci.Source, appDir, validator)
							if err != nil {
								// Empty map if source can't be loaded
								controlTitlesBySource[ci.Source] = make(map[string]string)
							} else {
								controlTitlesBySource[ci.Source] = loadedTitles
							}
						}
					}

					for _, ir := range ci.ImplementedRequirements {
						if ir.ControlId != "" {
							includeControls.Add(ir.ControlId)

							// Getting control title for id from map lookup
							if validator != nil {
								if _, exists := controlTitles[ir.ControlId]; !exists {
									// Get the title from the loaded source
									if title, found := controlTitlesBySource[ci.Source][ir.ControlId]; found {
										controlTitles[ir.ControlId] = title
									} else {
										// Empty string if title isn't available
										controlTitles[ir.ControlId] = ""
									}
								}
							} else {
								// Empty string if title isn't available
								controlTitles[ir.ControlId] = ""
							}
						}
					}

					// Process set parameters and extract values
					if ci.SetParameters != nil {
						for _, sp := range *ci.SetParameters {
							if sp.ParamId != "" && len(sp.Values) > 0 {
								setParameters[sp.ParamId] = sp.Values
							}
						}
					}
				}
			}
		}
	}

	controlIDs := includeControls.All()
	scope.IncludeControls = make([]ControlEntry, len(controlIDs))
	for i, id := range controlIDs {
		// Create parameter entries from current set parameter values
		var parameterSelections []ParameterEntry
		for paramID, values := range setParameters {
			// Use single value from set parameter entry to follow info command pattern
			paramValue := ""
			if len(values) > 0 {
				paramValue = values[0]
			}
			parameterSelections = append(parameterSelections, ParameterEntry{
				Name:  paramID, // Use parameter ID as name
				Value: paramValue,
			})
		}

		// If no specific parameters found, add *
		if len(parameterSelections) == 0 {
			parameterSelections = []ParameterEntry{{Name: "*", Value: "*"}}
		}

		scope.IncludeControls[i] = ControlEntry{
			ControlID:        id,
			ControlTitle:     controlTitles[id],
			IncludeRules:     []string{"*"}, // by default, include all rules
			SelectParameters: parameterSelections,
		}
	}
	sort.Slice(scope.IncludeControls, func(i, j int) bool {
		return scope.IncludeControls[i].ControlID < scope.IncludeControls[j].ControlID
	})

	return scope, nil
}

// ApplyScope alters the given OSCAL Assessment Plan based on the AssessmentScope.
func (a AssessmentScope) ApplyScope(assessmentPlan *oscalTypes.AssessmentPlan, logger hclog.Logger) {

	// This is a thin wrapper right now, but the goal to expand to different areas
	// of customization.
	a.applyControlScope(assessmentPlan, logger)
	a.applyRuleScope(assessmentPlan, logger)
}

// applyControlScope alters the AssessedControls of the given OSCAL Assessment Plan by the AssessmentScope
// IncludeControls.
func (a AssessmentScope) applyControlScope(assessmentPlan *oscalTypes.AssessmentPlan, logger hclog.Logger) {
	// "Any control specified within exclude-controls must first be within a range of explicitly
	// included controls, via include-controls or include-all."
	includedControls := includeControlsSet{}
	for _, entry := range a.IncludeControls {
		includedControls.Add(entry.ControlID)
	}
	logger.Debug("Found included controls", "count", len(includedControls))
	for _, controlT := range assessmentPlan.ReviewedControls.ControlSelections {
		if controlT.IncludeControls != nil {
			if controlT.Props != nil {
				for _, control := range *controlT.Props {
					// process control properties
					_ = control.Name
				}
			}
		}
	}
	if assessmentPlan.LocalDefinitions != nil {
		if assessmentPlan.LocalDefinitions.Activities != nil {
			for activityI := range *assessmentPlan.LocalDefinitions.Activities {
				activity := &(*assessmentPlan.LocalDefinitions.Activities)[activityI]
				if activity.RelatedControls != nil && activity.RelatedControls.ControlSelections != nil {
					controlSelections := activity.RelatedControls.ControlSelections
					for controlSelectionI := range controlSelections {
						controlSelection := &controlSelections[controlSelectionI]
						filterControlSelection(controlSelection, includedControls)
						if controlSelection.IncludeControls == nil {
							activity.RelatedControls = nil
							if activity.Props == nil {
								activity.Props = &[]oscalTypes.Property{}
							}
							skippedActivity := oscalTypes.Property{
								Name:  "skipped",
								Value: "true",
								Ns:    extensions.TrestleNameSpace,
							}
							*activity.Props = append(*activity.Props, skippedActivity)
						}
					}
				}
			}
		}
	}
	if assessmentPlan.ReviewedControls.ControlSelections != nil {
		for controlSelectionI := range assessmentPlan.ReviewedControls.ControlSelections {
			controlSelection := &assessmentPlan.ReviewedControls.ControlSelections[controlSelectionI]
			filterControlSelection(controlSelection, includedControls)
		}
	}
}

// applyRuleScope alters the AssessedControls of activities based on IncludeRules and ExcludeRules configuration
func (a AssessmentScope) applyRuleScope(assessmentPlan *oscalTypes.AssessmentPlan, logger hclog.Logger) {
	// Convert global exclude rules to a map for fast lookup
	globalExcludeRules := make(map[string]struct{})
	for _, rule := range a.GlobalExcludeRules {
		globalExcludeRules[rule] = struct{}{}
	}

	// Check if globalExcludeRules contains "*" (exclude all rules globally)
	_, hasGlobalWildcard := globalExcludeRules["*"]
	if hasGlobalWildcard {
		logger.Warn("Global exclude rules contains '*' - all rules will be excluded from all controls")
	}

	// Build a map of control ID to ControlEntry for quick lookup
	controlRuleConfig := make(map[string]ControlEntry)
	for _, entry := range a.IncludeControls {
		// Normalize the entry before storing it in the map
		normalizedEntry := entry

		// If globalExcludeRules contains "*", all rules are globally excluded
		if hasGlobalWildcard {
			normalizedEntry.IncludeRules = []string{} // Clear includeRules since all rules are globally excluded
			normalizedEntry.ExcludeRules = []string{} // Clear excludeRules since global takes precedence
		} else if a.isRuleInList("*", normalizedEntry.ExcludeRules) {
			// If excludeRules contains "*", includeRules doesn't matter
			normalizedEntry.IncludeRules = []string{} // Clear includeRules since they don't matter
		} else if len(normalizedEntry.IncludeRules) == 0 {
			normalizedEntry.IncludeRules = []string{"*"}
		}
		controlRuleConfig[entry.ControlID] = normalizedEntry
	}
	logger.Debug("Applying rule scope filtering", "globalExcludeRules", len(globalExcludeRules))

	if assessmentPlan.LocalDefinitions != nil {
		if assessmentPlan.LocalDefinitions.Activities != nil {
			for activityI := range *assessmentPlan.LocalDefinitions.Activities {
				activity := &(*assessmentPlan.LocalDefinitions.Activities)[activityI]

				// If the activity has no title it cannot be mapped to a rule
				if activity.Title == "" {
					logger.Debug("Activity is missing title, skipping", "activity", activity.UUID)
					continue
				}

				if activity.RelatedControls != nil && activity.RelatedControls.ControlSelections != nil {
					controlSelections := activity.RelatedControls.ControlSelections
					for controlSelectionI := range controlSelections {
						controlSelection := &controlSelections[controlSelectionI]
						a.filterControlSelectionByRule(controlSelection, activity.Title, controlRuleConfig, globalExcludeRules, logger, activity.Title)
						if controlSelection.IncludeControls == nil {
							activity.RelatedControls = nil
							if activity.Props == nil {
								activity.Props = &[]oscalTypes.Property{}
							}
							skippedActivity := oscalTypes.Property{
								Name:  "skipped",
								Value: "true",
								Ns:    extensions.TrestleNameSpace,
							}
							*activity.Props = append(*activity.Props, skippedActivity)
						}
					}
				}

				if activity.Steps != nil {
					for stepI := range *activity.Steps {
						step := &(*activity.Steps)[stepI]
						if step.ReviewedControls == nil {
							continue
						}
						if step.ReviewedControls.ControlSelections == nil {
							continue
						}
						controlSelections := step.ReviewedControls.ControlSelections
						for controlSelectionI := range controlSelections {
							controlSelection := &controlSelections[controlSelectionI]
							a.filterControlSelectionByRule(controlSelection, activity.Title, controlRuleConfig, globalExcludeRules, logger, activity.Title)
							if controlSelection.IncludeControls == nil {
								activity.RelatedControls.ControlSelections = nil
								step.ReviewedControls = nil
								if step.Props == nil {
									step.Props = &[]oscalTypes.Property{}
								}
								skipped := oscalTypes.Property{
									Name:  "skipped",
									Value: "true",
									Ns:    extensions.TrestleNameSpace,
								}
								*step.Props = append(*step.Props, skipped)
							}
						}
					}
				}
			}
		}
	}
}
func filterControlSelection(controlSelection *oscalTypes.AssessedControls, includedControls includeControlsSet) {
	// The new included controls should be the intersection of
	// the originally included controls and the newly included controls.
	// ExcludedControls are preserved.

	// includedControls specifies everything we allow - do not include all
	includedAll := controlSelection.IncludeAll != nil
	controlSelection.IncludeAll = nil

	originalIncludedControls := includeControlsSet{}
	if controlSelection.IncludeControls != nil {
		for _, controlId := range *controlSelection.IncludeControls {
			originalIncludedControls.Add(controlId.ControlId)
		}
	}
	var newIncludedControls []oscalTypes.AssessedControlsSelectControlById
	for controlId := range includedControls {
		if includedAll || originalIncludedControls.Has(controlId) {
			newIncludedControls = append(newIncludedControls, oscalTypes.AssessedControlsSelectControlById{
				ControlId: controlId,
			})
		}
	}
	if newIncludedControls != nil {
		// Sort newIncludedControls by ControlId to ensure consistency after filtering
		sort.Slice(newIncludedControls, func(i, j int) bool {
			return newIncludedControls[i].ControlId < newIncludedControls[j].ControlId
		})
		controlSelection.IncludeControls = &newIncludedControls
	} else {
		controlSelection.IncludeControls = nil
	}
}

// filterControlSelectionByRule removes controls from a selection if the activity's rule should be excluded for those controls
func (a AssessmentScope) filterControlSelectionByRule(controlSelection *oscalTypes.AssessedControls, activityRuleID string, controlRuleConfig map[string]ControlEntry, globalExcludeRules map[string]struct{}, logger hclog.Logger, activityTitle string) {
	if controlSelection.IncludeControls == nil {
		logger.Debug("No controls to filter for activity", "activity", activityTitle)
		return
	}

	var filteredControls []oscalTypes.AssessedControlsSelectControlById

	for _, control := range *controlSelection.IncludeControls {
		controlEntry, exists := controlRuleConfig[control.ControlId]
		if !exists {
			// Control not in our scope configuration, keep it
			filteredControls = append(filteredControls, control)
			continue
		}

		shouldKeepControl := true

		// Check global exclude rules first (highest priority)
		if _, isGloballyExcluded := globalExcludeRules[activityRuleID]; isGloballyExcluded {
			shouldKeepControl = false
			logger.Debug("Removing control from activity due to globally excluded rule", "control", control.ControlId, "rule", activityRuleID)
		} else if _, isAllGloballyExcluded := globalExcludeRules["*"]; isAllGloballyExcluded {
			// Check if "*" is globally excluded (exclude all rules)
			shouldKeepControl = false
			logger.Debug("Removing control from activity due to global exclude all rules", "control", control.ControlId, "rule", activityRuleID)
		} else if a.isRuleInList(activityRuleID, controlEntry.ExcludeRules) {
			// Check if rule is in control-specific exclude list
			shouldKeepControl = false
			logger.Debug("Removing control from activity due to control-specific excluded rule", "control", control.ControlId, "rule", activityRuleID)
		} else if !a.isRuleInList("*", controlEntry.IncludeRules) && !a.isRuleInList(activityRuleID, controlEntry.IncludeRules) {
			// Check if rule should be included (if not using wildcard)
			shouldKeepControl = false
			logger.Debug("Removing control from activity due to rule not in include list", "control", control.ControlId, "rule", activityRuleID)
		}

		if shouldKeepControl {
			filteredControls = append(filteredControls, control)
		}
	}

	// Update the control selection
	if len(filteredControls) == 0 {
		controlSelection.IncludeControls = nil
	} else {
		*controlSelection.IncludeControls = filteredControls
	}
}

// isRuleInList checks if a rule ID exists in a list of rules
func (a AssessmentScope) isRuleInList(ruleID string, ruleList []string) bool {
	for _, rule := range ruleList {
		if rule == ruleID {
			return true
		}
	}
	return false
}

type includeControlsSet map[string]struct{}

func (i includeControlsSet) Add(controlID string) {
	i[controlID] = struct{}{}
}

func (i includeControlsSet) All() []string {
	keys := make([]string, 0, len(i))
	for controlId := range i {
		keys = append(keys, controlId)
	}
	return keys
}

func (i includeControlsSet) Has(controlID string) bool {
	_, found := i[controlID]
	return found
}
