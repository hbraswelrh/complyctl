// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"fmt"
	"sort"
	"strings"

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

// componentDefResult holds the results from processing component definitions
type componentDefResult struct {
	includeControls   includeControlsSet
	controlTitles     map[string]string
	controlParameters map[string]map[string][]string
}

// NewAssessmentScopeFromCDs creates and populates an AssessmentScope struct for a given framework id and set of
// OSCAL Component Definitions.
func NewAssessmentScopeFromCDs(frameworkId string, appDir ApplicationDirectory, validator validation.Validator, cds ...oscalTypes.ComponentDefinition) (AssessmentScope, error) {
	scope := NewAssessmentScope(frameworkId)

	if cds == nil {
		return AssessmentScope{}, fmt.Errorf("no component definitions found")
	}

	// Initialize processing results
	result := &componentDefResult{
		includeControls:   make(includeControlsSet),
		controlTitles:     make(map[string]string),
		controlParameters: make(map[string]map[string][]string),
	}

	// Map to store control titles by source to avoid loading the same source multiple times
	controlTitlesBySource := make(map[string]map[string]string)

	// Process control implementations and build control relationships
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

				if ci.Props == nil {
					continue
				}

				frameworkProp, found := extensions.GetTrestleProp(extensions.FrameworkProp, *ci.Props)
				if !found || frameworkProp.Value != frameworkId {
					continue
				}

				// Load control titles from source on control implementation if needed
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

				// Process implemented requirements
				for _, ir := range ci.ImplementedRequirements {
					if ir.ControlId != "" {
						result.includeControls.Add(ir.ControlId)

						// Set control title if not already set
						if _, exists := result.controlTitles[ir.ControlId]; !exists {
							if validator != nil {
								// Get the title from the loaded source
								if title, found := controlTitlesBySource[ci.Source][ir.ControlId]; found {
									result.controlTitles[ir.ControlId] = title
								} else {
									// Empty string if title isn't available
									result.controlTitles[ir.ControlId] = ""
								}
							} else {
								// Empty string if the title isn't available
								result.controlTitles[ir.ControlId] = ""
							}
						}
					}
				}

				// Process set parameters
				if ci.SetParameters != nil {
					processSetParameters(ci, result, cds)
				}
			}
		}
	}

	// Build control entries from extracted data
	scope.IncludeControls = buildControlEntries(result)

	return scope, nil
}

// processSetParameters processes set parameters for a control implementation
func processSetParameters(ci oscalTypes.ControlImplementationSet, result *componentDefResult, cds []oscalTypes.ComponentDefinition) {
	implementedSetParams := make(map[string][]string)
	for _, sp := range *ci.SetParameters {
		if sp.ParamId != "" && len(sp.Values) > 0 {
			implementedSetParams[sp.ParamId] = sp.Values
		}
	}

	remarksProps := extractRemarksProperties(cds)

	for _, ir := range ci.ImplementedRequirements {
		if ir.ControlId != "" && ir.Props != nil {
			controlRules := make(map[string]bool)
			for _, prop := range *ir.Props {
				if prop.Name == extensions.RuleIdProp {
					controlRules[prop.Value] = true
				}
			}

			for _, props := range remarksProps {
				var ruleID string
				var parametersInGroup []string

				for _, prop := range props {
					if prop.Name == extensions.RuleIdProp {
						ruleID = prop.Value
					}
					if isParameterIdProperty(prop.Name) {
						parametersInGroup = append(parametersInGroup, prop.Value)
					}
				}

				if ruleID != "" && controlRules[ruleID] {
					for _, paramID := range parametersInGroup {
						if paramValues, hasSetParam := implementedSetParams[paramID]; hasSetParam {
							if result.controlParameters[ir.ControlId] == nil {
								result.controlParameters[ir.ControlId] = make(map[string][]string)
							}
							result.controlParameters[ir.ControlId][paramID] = paramValues
						}
					}
				}
			}
		}
	}
}

// buildControlEntries builds the final control entries from the processing results
func buildControlEntries(result *componentDefResult) []ControlEntry {
	controlIDs := result.includeControls.All()
	controlEntries := make([]ControlEntry, len(controlIDs))

	for i, id := range controlIDs {
		var parameterSelections []ParameterEntry
		if controlParams, exists := result.controlParameters[id]; exists {
			for paramID, values := range controlParams {
				paramValue := ""
				if len(values) > 0 {
					paramValue = values[0]
				}

				parameterSelections = append(parameterSelections, ParameterEntry{
					Name:  paramID,
					Value: paramValue,
				})
			}
		}

		if len(parameterSelections) == 0 {
			parameterSelections = []ParameterEntry{{Name: "N/A", Value: "N/A"}}
		}

		controlEntries[i] = ControlEntry{
			ControlID:        id,
			ControlTitle:     result.controlTitles[id],
			IncludeRules:     []string{"*"}, // by default, include all rules
			SelectParameters: parameterSelections,
		}
	}

	sort.Slice(controlEntries, func(i, j int) bool {
		return controlEntries[i].ControlID < controlEntries[j].ControlID
	})

	return controlEntries
}

// ApplyScope alters the given OSCAL Assessment Plan based on the AssessmentScope.
// If componentDefs is provided, it will be used for parameter validation; otherwise validation is skipped.
func (a AssessmentScope) ApplyScope(assessmentPlan *oscalTypes.AssessmentPlan, logger hclog.Logger, componentDefs ...oscalTypes.ComponentDefinition) error {

	// This is a thin wrapper right now, but the goal to expand to different areas
	// of customization.
	a.applyControlScope(assessmentPlan, logger)
	a.applyRuleScope(assessmentPlan, logger)
	return a.applyParameterScope(assessmentPlan, componentDefs, logger)
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

// applyParameterScope updates activity properties based on SelectParameters values
// If componentDefs is provided, validates against component definitions; otherwise validates against assessment plan
func (a AssessmentScope) applyParameterScope(assessmentPlan *oscalTypes.AssessmentPlan, componentDefs []oscalTypes.ComponentDefinition, logger hclog.Logger) error {
	// Build map of control ID to parameters for that control
	controlParams := make(map[string]map[string]string)

	for _, entry := range a.IncludeControls {
		if len(entry.SelectParameters) > 0 {
			controlParams[entry.ControlID] = make(map[string]string)
			for _, p := range entry.SelectParameters {
				if p.Name == "" {
					continue
				}
				controlParams[entry.ControlID][p.Name] = p.Value
			}
		}
	}

	if len(controlParams) == 0 {
		return nil
	}

	if assessmentPlan.LocalDefinitions == nil || assessmentPlan.LocalDefinitions.Activities == nil {
		return nil
	}

	// Validate all selected parameters by control before updating the assessment plan
	if err := a.validateAllControlParameterSelections(controlParams, componentDefs); err != nil {
		logger.Debug("Parameter validation failed", "error", err)
		return err // Return immediately without modifying the assessment plan.
	}

	logger.Debug("Parameter validation passed, applying parameter updates")

	// Update activity parameters based on control relationships
	for activityI := range *assessmentPlan.LocalDefinitions.Activities {
		activity := &(*assessmentPlan.LocalDefinitions.Activities)[activityI]
		if activity.Props == nil {
			continue
		}

		logger.Debug("Scoping activity parameters", "activity", activity.Title)

		// Get control IDs related to this activity
		relatedControlIDs := a.getRelatedControlIDs(activity)

		// Apply parameters for the related controls
		props := *activity.Props
		for i := range props {
			if props[i].Class == extensions.TestParameterClass {
				var newValue string

				if len(relatedControlIDs) > 0 {
					// Find the appropriate parameter value from the related controls
					newValue = a.findParameterValueForControls(props[i].Name, relatedControlIDs, controlParams)
				}

				if newValue != "" {
					props[i].Value = newValue
				}
			}
		}
		*activity.Props = props
	}
	return nil
}

// getRelatedControlIDs extracts control IDs from an activity's RelatedControls
func (a AssessmentScope) getRelatedControlIDs(activity *oscalTypes.Activity) []string {
	var controlIDs []string

	if activity.RelatedControls == nil || activity.RelatedControls.ControlSelections == nil {
		return controlIDs
	}

	for _, controlSelection := range activity.RelatedControls.ControlSelections {
		if controlSelection.IncludeControls != nil {
			for _, control := range *controlSelection.IncludeControls {
				controlIDs = append(controlIDs, control.ControlId)
			}
		}
	}

	return controlIDs
}

// findParameterValueForControls finds the parameter value from related controls.
func (a AssessmentScope) findParameterValueForControls(paramName string, relatedControlIDs []string, controlParams map[string]map[string]string) string {
	// Check each related control in order for the parameter
	for _, controlID := range relatedControlIDs {
		if params, ok := controlParams[controlID]; ok {
			if value, exists := params[paramName]; exists {
				return value
			}
		}
	}

	return "" // No value found for this parameter in any related control
}

// validateAllControlParameterSelections validates parameter selections
func (a AssessmentScope) validateAllControlParameterSelections(controlParams map[string]map[string]string, componentDefs []oscalTypes.ComponentDefinition) error {
	remarksProps := extractRemarksProperties(componentDefs)
	var validationErrors []string

	for controlID, params := range controlParams {
		for paramID, selectedValue := range params {
			isValid, availableAlternatives := filterParameterSelection(paramID, selectedValue, remarksProps)
			if !isValid {
				errorMsg := fmt.Sprintf("control '%s': parameter '%s' has invalid value '%s'. Available alternatives: [%s]",
					controlID, paramID, selectedValue, strings.Join(availableAlternatives, ", "))
				validationErrors = append(validationErrors, errorMsg)
			}
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("parameter validation failed:\n%s", strings.Join(validationErrors, "\n"))
	}

	return nil
}

// extractRemarksProperties extracts remarks-grouped properties
func extractRemarksProperties(componentDefs []oscalTypes.ComponentDefinition) map[string][]oscalTypes.Property {
	remarksProps := make(map[string][]oscalTypes.Property)

	for _, compDef := range componentDefs {
		if compDef.Components == nil {
			continue
		}
		for _, component := range *compDef.Components {
			if component.Props != nil {
				for _, prop := range *component.Props {
					if prop.Remarks != "" {
						remarksProps[prop.Remarks] = append(remarksProps[prop.Remarks], prop)
					}
				}
			}
		}
	}

	return remarksProps
}

// ValidateParameterValue validates a parameter value to configure the assessment plan.
func ValidateParameterValue(parameterID, selectedValue string, componentDefinitions []oscalTypes.ComponentDefinition) error {
	remarksProps := extractRemarksProperties(componentDefinitions)
	isValid, availableAlternatives := filterParameterSelection(parameterID, selectedValue, remarksProps)

	if !isValid {
		return fmt.Errorf("parameter '%s' has invalid value '%s'. Available alternatives: [%s]",
			parameterID, selectedValue, strings.Join(availableAlternatives, ", "))
	}

	return nil
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

// filterParameterSelection validates a parameter selection against alternatives
func filterParameterSelection(parameterID, selectedValue string, remarksProps map[string][]oscalTypes.Property) (bool, []string) {
	if selectedValue == "" || selectedValue == "N/A" {
		return true, nil
	}

	var allPossibleAlternatives []string
	var hasAlternatives bool

	for _, props := range remarksProps {
		var foundParameterID bool
		var parameterSuffix string

		// Look for the parameter ID in this remarks group to get its suffix
		for _, prop := range props {
			if prop.Value == parameterID && isParameterIdProperty(prop.Name) {
				foundParameterID = true
				if prop.Name == extensions.ParameterIdProp {
					parameterSuffix = ""
				} else {
					parameterSuffix = strings.TrimPrefix(prop.Name, extensions.ParameterIdProp+"_")
				}
				break
			}
		}

		if foundParameterID {
			var alternativesPropertyName string
			if parameterSuffix == "" {
				alternativesPropertyName = "Parameter_Value_Alternatives"
			} else {
				alternativesPropertyName = "Parameter_Value_Alternatives_" + parameterSuffix
			}

			for _, prop := range props {
				if prop.Name == alternativesPropertyName {
					alternatives := parseParameterAlternatives(prop.Value)
					if len(alternatives) > 0 {
						hasAlternatives = true
						for _, altValue := range alternatives {
							if altValue == selectedValue {
								return true, alternatives
							}
						}
						allPossibleAlternatives = append(allPossibleAlternatives, alternatives...)
					}
					break
				}
			}
		}
	}

	if hasAlternatives {
		cleanedAlternatives := removeDuplicates(allPossibleAlternatives)
		return false, cleanedAlternatives
	}

	return true, nil
}

// isParameterIdProperty checks if a property matches Parameter_Id.
func isParameterIdProperty(propertyName string) bool {
	return propertyName == extensions.ParameterIdProp || strings.HasPrefix(propertyName, extensions.ParameterIdProp+"_")
}

// parseParameterAlternatives parses the Parameter_Value_Alternatives value to extract choice options
// Returns the keys that users can input as alternatives.
func parseParameterAlternatives(alternativesValue string) []string {
	var alternatives []string

	cleaned := strings.Trim(alternativesValue, "\"'")
	cleaned = strings.Trim(cleaned, "{}")

	if cleaned == "" {
		return alternatives
	}

	// Split by comma and extract keys
	pairs := strings.Split(cleaned, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) == 2 {
			key := strings.Trim(strings.TrimSpace(parts[0]), "\"'")
			if key != "" {
				alternatives = append(alternatives, key)
			}
		}
	}
	return removeDuplicates(alternatives)
}

// removeDuplicates removes duplicate items from a slice
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, val := range slice {
		if _, ok := seen[val]; !ok {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
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
