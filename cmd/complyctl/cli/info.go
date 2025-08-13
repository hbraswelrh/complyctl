// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/oscal-compass/oscal-sdk-go/extensions"
	"github.com/oscal-compass/oscal-sdk-go/models/components"
	"github.com/oscal-compass/oscal-sdk-go/settings"
	"github.com/oscal-compass/oscal-sdk-go/validation"

	"github.com/spf13/cobra"

	"github.com/complytime/complyctl/cmd/complyctl/option"
	"github.com/complytime/complyctl/internal/complytime"
	"github.com/complytime/complyctl/internal/terminal"
)

const (
	colWidthControlID      = 12
	colWidthControlTitle   = 40
	colWidthImplStatus     = 21
	colWidthPluginsUsed    = 24
	colWidthRulesInControl = 55
	colWidthPluginUsed     = 31
)

var (
	tableHeaderStyle = table.DefaultStyles().Header.
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")). // Light grey border
				BorderBottom(true).
				Bold(false)

	tableCellStyle = table.DefaultStyles().Cell.
			Foreground(lipgloss.Color("250")). // Very light grey text
			Bold(true)

	// Style for key-value pair keys
	keyStyle = lipgloss.NewStyle().
			PaddingRight(1)

	// Style for key-value pairs values
	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Bold(true)

	// Container style for key-value information blocks
	infoContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")). // Light grey border for containers
				Padding(0, 1).
				Width(90) // Fixed width for consistent formatting
)

// parameter represents details about a parameter for easy mapping of parameters to set values.
type parameter struct {
	ID          string // Name of param
	Description string
	Values      []string // Current set value(s) - used for rule display
}

// rule represents details about a rule for easy mapping of rules to plugins.
type rule struct {
	ID          string
	Plugin      string
	Description string
	Parameters  []parameter
}

// control repsents details about a control across component sources.
type control struct {
	ID                   string
	Title                string
	Description          string
	ImplementationStatus string
	Rules                []rule
}

// rulePluginMap maps a Rule ID to the plugin that implements it.
type rulePluginMap map[string]string

// ruleRemarksMap maps a Rule ID to its associated remarks value.
type ruleRemarksMap map[string]string

// remarksPropertiesMap maps a remarks value to a list of properties.
type remarksPropertiesMap map[string][]oscalTypes.Property

// indexedControls holds extracted details for Controls, indexed by Control ID.
type indexedControls map[string]control

// indexedSetParameters maps a Parameter ID to its list of set values.
type indexedSetParameters map[string][]string

type infoOptions struct {
	*option.Common
	complyTimeOpts *option.ComplyTime
	controlID      string // show info for a specific control ID
	ruleID         string // show info for a specific rule ID
	parameterID    string // show info for a specific parameter ID
	limit          int    // limit number for table rows shown in terminal
	plain          bool   // print plain table only
}

func infoCmd(common *option.Common) *cobra.Command {
	infoOpts := &infoOptions{
		Common:         common,
		complyTimeOpts: &option.ComplyTime{},
	}
	cmd := &cobra.Command{
		Use:     "info <framework-id> [flags]",
		Short:   "Show information about a framework's controls and rules",
		Example: " complyctl info anssi_bp28_minimal\n complyctl info anssi_bp28_minimal --control r31\n complyctl info anssi_bp28_minimal --rule enable_authselect\n complyctl info anssi_bp28_minimal --parameter var_accounts_password_minlen_login_defs",
		Args:    cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				infoOpts.complyTimeOpts.FrameworkID = filepath.Clean(args[0])
			}
		},
		RunE: func(_ *cobra.Command, _ []string) error { return runInfo(infoOpts) },
	}
	cmd.Flags().StringVarP(&infoOpts.controlID, "control", "c", "", "show info for a specific control ID")
	cmd.Flags().StringVarP(&infoOpts.ruleID, "rule", "r", "", "show info for a specific rule ID")
	cmd.Flags().StringVarP(&infoOpts.parameterID, "parameter", "P", "", "show info for a specific parameter ID")
	cmd.Flags().IntVarP(&infoOpts.limit, "limit", "l", 0, "limit the number of table rows")
	cmd.Flags().BoolVarP(&infoOpts.plain, "plain", "p", false, "print the table with minimal formatting")
	infoOpts.complyTimeOpts.BindFlags(cmd.Flags())
	return cmd
}

// runInfo executes the info command using the provided options.
func runInfo(opts *infoOptions) error {

	appDir, err := complytime.NewApplicationDirectory(true)
	if err != nil {
		return fmt.Errorf("failed to initialize application directory: %w", err)
	}
	logger.Debug(fmt.Sprintf("Using application directory: %s", appDir.AppDir()))

	validator := validation.NewSchemaValidator()

	compDefs, err := complytime.FindComponentDefinitions(appDir.BundleDir(), validator)
	if err != nil {
		return fmt.Errorf("failed to find component definitions: %w", err)
	}

	frameworkComponents, validationComponents := loadComponents(compDefs, opts.complyTimeOpts.FrameworkID)
	if len(frameworkComponents) == 0 {
		return fmt.Errorf("no components found for framework ID '%s'", opts.complyTimeOpts.FrameworkID)
	}

	rulePlugins := extractRulePluginMapping(validationComponents)

	ruleRemarks, remarksProps := processComponentProperties(frameworkComponents)

	indexedControls, indexedSetParameters := processControlImplementations(frameworkComponents, rulePlugins, appDir, validator)

	// Display info based on controlID, ruleID, or parameterID flag being passed at CLI
	if opts.controlID != "" {
		return displayControlInfo(opts, indexedControls)
	} else if opts.ruleID != "" {
		return displayRuleInfo(opts, opts.ruleID, ruleRemarks, remarksProps, indexedSetParameters)
	} else if opts.parameterID != "" {
		return displayParameterInfo(opts, opts.parameterID, indexedSetParameters, ruleRemarks, remarksProps, appDir, validator)
	} else {
		return displayAllControls(opts, indexedControls)
	}
}

// extractRulePluginMapping builds map of rule -> plugins from a list of validation components.
func extractRulePluginMapping(validationComponents []oscalTypes.DefinedComponent) rulePluginMap {
	var pluginsForRule = make(rulePluginMap)

	for _, comp := range validationComponents {
		for _, prop := range *comp.Props {
			if prop.Name == extensions.RuleIdProp {
				pluginsForRule[prop.Value] = comp.Title
			}
		}
	}
	return pluginsForRule
}

// processControlImplementations extracts control details and set parameters from component definitions.
func processControlImplementations(components []oscalTypes.DefinedComponent, rulePluginsMap rulePluginMap, appDir complytime.ApplicationDirectory, validator *validation.SchemaValidator) (indexedControls, indexedSetParameters) {
	controlMap := make(indexedControls)
	setParameters := make(indexedSetParameters)

	for _, comp := range components {

		if comp.ControlImplementations == nil {
			continue
		}

		for _, controlImp := range *comp.ControlImplementations {
			if controlImp.ImplementedRequirements != nil {
				for _, ir := range controlImp.ImplementedRequirements {
					controlDetails, ok := controlMap[ir.ControlId]
					if !ok {
						// Initialize controlDetails if not already present
						controlTitle, err := complytime.GetControlTitle(ir.ControlId, controlImp.Source, appDir, validator)
						if err != nil {
							logger.Warn("could not get title for control %s: %v", ir.ControlId, err)
							controlTitle = ""
						}

						controlDetails = control{
							ID:          ir.ControlId,
							Title:       controlTitle,
							Description: ir.Description,
							Rules:       []rule{},
						}
					}

					if ir.Props != nil {
						for _, p := range *ir.Props {
							switch p.Name {
							case extensions.RuleIdProp:
								rule := extractRuleDetails(*ir.Props)
								rule.ID = p.Value
								rule.Plugin = rulePluginsMap[rule.ID]
								controlDetails.Rules = append(controlDetails.Rules, rule)
							case "implementation-status":
								controlDetails.ImplementationStatus = p.Value
							}
						}
					}
					controlMap[ir.ControlId] = controlDetails // Update map with potentially modified control
				}
			}

			// Process set parameters
			if controlImp.SetParameters != nil {
				for _, sp := range *controlImp.SetParameters {
					if sp.ParamId != "" && len(sp.Values) > 0 {
						setParameters[sp.ParamId] = sp.Values
					}
				}
			}
		}
	}
	return controlMap, setParameters
}

// processComponentProperties extracts rule and property information from
// component definitions into indexed maps.
func processComponentProperties(compDefs []oscalTypes.DefinedComponent) (ruleRemarksMap, remarksPropertiesMap) {
	ruleRemarksMap := make(ruleRemarksMap)
	remarksPropsMap := make(remarksPropertiesMap)

	for _, comp := range compDefs {
		if comp.Props == nil {
			continue
		}
		for _, prop := range *comp.Props {
			remarksPropsMap[prop.Remarks] = append(remarksPropsMap[prop.Remarks], prop)
			if prop.Name == "Rule_Id" && prop.Value != "" && prop.Remarks != "" {
				ruleRemarksMap[prop.Value] = prop.Remarks
			}
		}
	}
	return ruleRemarksMap, remarksPropsMap
}

// findRulesUsingParameter finds all rules that use a specific parameter.
func findRulesUsingParameter(parameterID string, ruleRemarks ruleRemarksMap, remarksProps remarksPropertiesMap) []string {
	var rulesUsingParam []string

	for _, props := range remarksProps {
		hasParameter := false
		var ruleID string

		for _, prop := range props {
			// Check for Parameter_Id patterns
			if isParameterIdProperty(prop.Name) && prop.Value == parameterID {
				hasParameter = true
			}
			if prop.Name == "Rule_Id" {
				ruleID = prop.Value
			}
		}

		if hasParameter && ruleID != "" {
			rulesUsingParam = append(rulesUsingParam, ruleID)
		}
	}

	return removeDuplicates(rulesUsingParam)
}

// loadComponents retrieves components from component definitions by framework ID.
func loadComponents(componentDefinitions []oscalTypes.ComponentDefinition, frameworkID string) ([]oscalTypes.DefinedComponent, []oscalTypes.DefinedComponent) {

	var frameworkComponents []oscalTypes.DefinedComponent
	var validationComponents []oscalTypes.DefinedComponent

	for _, compDef := range componentDefinitions {
		if compDef.Components == nil {
			continue
		}

		for _, component := range *compDef.Components {
			if component.Type == string(components.Validation) {
				validationComponents = append(validationComponents, component)
			} else {
				if component.ControlImplementations == nil {
					continue
				}
				for _, controlImp := range *component.ControlImplementations {
					framework, ok := settings.GetFrameworkShortName(controlImp)
					if ok && framework == frameworkID {
						frameworkComponents = append(frameworkComponents, component)
						break // Component belongs to this framework, move to next component
					}
				}
			}
		}
	}
	return frameworkComponents, validationComponents
}

// extractRuleDetails parses properties to fill a RuleInfo struct.
func extractRuleDetails(props []oscalTypes.Property) rule {
	info := rule{}
	for _, prop := range props {
		switch prop.Name {
		case "Rule_Description":
			info.Description = prop.Value
		case "Parameter_Id":
			param := parameter{
				ID: prop.Value,
			}
			// Look for description in the same property set
			for _, otherProp := range props {
				if otherProp.Name == extensions.ParameterDescriptionProp && otherProp.Remarks == prop.Remarks {
					param.Description = otherProp.Value
					break
				}
			}
			info.Parameters = append(info.Parameters, param)
		}
	}
	return info
}

// getParameterDetailsColumnsAndRows prepares columns and rows for the parameter details table.
func getParameterDetailsColumnsAndRows(paramDetails parameter, paramValues []string, ruleRemarks ruleRemarksMap, remarksProps remarksPropertiesMap, appDir complytime.ApplicationDirectory, validator *validation.SchemaValidator, frameworkID string) ([]table.Column, []table.Row) {
	var rows []table.Row

	currentValue := getParameterCurrentValue(paramValues)
	// Use remarks-based approach to find parameter alternatives
	availableAlternatives := findParameterAlternativesFromRemarks(paramDetails.ID, remarksProps)

	// Style for bold current value
	boldValueStyle := lipgloss.NewStyle().Bold(true)

	if len(availableAlternatives) > 0 {
		// First, add the current value row (bold and first)
		if currentValue != "" {
			rows = append(rows, table.Row{boldValueStyle.Render(currentValue), boldValueStyle.Render("Current")})
		}

		// Then add alternatives (excluding the current value to avoid duplicates)
		for _, alternative := range availableAlternatives {
			if alternative != currentValue {
				rows = append(rows, table.Row{alternative, "Alternative"})
			}
		}
	} else {
		// If no alternatives available, show current value (bold)
		if currentValue != "" {
			rows = append(rows, table.Row{boldValueStyle.Render(currentValue), boldValueStyle.Render("Set")})
		}
	}

	columns := []table.Column{
		{Title: "Parameter Value(s)", Width: 20},
		{Title: "Status", Width: 15},
	}

	return calculateDynamicColumnWidths(columns, rows), rows
}

// getParameterRulesColumnsAndRows prepares columns and rows for the parameter rules table.
func getParameterRulesColumnsAndRows(parameterID string, ruleRemarks ruleRemarksMap, remarksProps remarksPropertiesMap) ([]table.Column, []table.Row) {
	var rows []table.Row

	usedByRules := findRulesUsingParameter(parameterID, ruleRemarks, remarksProps)

	// Create rows for each rule
	for _, rule := range usedByRules {
		rows = append(rows, table.Row{rule})
	}

	columns := []table.Column{
		{Title: "Rules Using This Parameter", Width: 50},
	}

	return calculateDynamicColumnWidths(columns, rows), rows
}

// findParameterAlternativesFromRemarks finds parameter alternatives using the remarks-grouped properties.
func findParameterAlternativesFromRemarks(parameterID string, remarksProps remarksPropertiesMap) []string {
	// Find the remarks group of the parameter
	var parameterRemarks string
	for remarks, props := range remarksProps {
		for _, prop := range props {
			// Check for Parameter_Id patterns
			if isParameterIdProperty(prop.Name) && prop.Value == parameterID {
				parameterRemarks = remarks
				break
			}
		}
		if parameterRemarks != "" {
			break
		}
	}

	// If the parameter's remarks group is found, check for alternatives in same group
	if parameterRemarks != "" {
		if props, ok := remarksProps[parameterRemarks]; ok {
			for _, prop := range props {
				if strings.HasPrefix(prop.Name, "Parameter_Value_Alternatives_") {
					return parseParameterAlternatives(prop.Value)
				}
			}
		}
	}

	return []string{}
}

// runBubbleTeaProgram executes the model
func runBubbleTeaProgram(model terminal.Model, output io.Writer) error {

	if _, err := tea.NewProgram(model, tea.WithOutput(output)).Run(); err != nil {
		return fmt.Errorf("failed to display information: %w", err)
	}
	return nil
}

// removeDuplicates removes duplicate items from a slice
func removeDuplicates[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := []T{}

	for _, val := range slice {
		if _, ok := seen[val]; !ok {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

// isParameterIdProperty checks if a property name matches Parameter_Id pattern using OSCAL SDK extensions.
func isParameterIdProperty(propName string) bool {
	// Handles Parameter_Id patterns
	if propName == extensions.ParameterIdProp {
		return true
	}
	// Check for suffix patterns
	return strings.HasPrefix(propName, extensions.ParameterIdProp+"_")
}

// isParameterDescriptionProperty checks if a property name matches Parameter_Description pattern using OSCAL SDK extensions.
func isParameterDescriptionProperty(propName string) bool {
	// Handles Parameter_Description patterns
	if propName == extensions.ParameterDescriptionProp {
		return true
	}
	// Check for suffix patterns
	return strings.HasPrefix(propName, extensions.ParameterDescriptionProp+"_")
}

// renderKeyValuePair renders a single key-value pair using predefined lipgloss styles.
func renderKeyValuePair(key, value string) string {
	return keyStyle.Render(key) + ": " + valueStyle.Render(value)
}

// getControlRulesColumnsAndRows prepares columns and rows for a specific control's rules table.
func getControlRulesColumnsAndRows(control control) ([]table.Column, []table.Row) {
	// Sort rules by ID for logical ordering in the table
	sort.Slice(control.Rules, func(i, j int) bool {
		return control.Rules[i].ID < control.Rules[j].ID
	})

	var rows []table.Row
	for _, rule := range control.Rules {
		row := table.Row{rule.ID, rule.Plugin}
		rows = append(rows, row)
	}

	columns := []table.Column{
		{Title: "Rules In Control", Width: colWidthRulesInControl},
		{Title: "Plugin Used", Width: colWidthPluginUsed},
	}

	return calculateDynamicColumnWidths(columns, rows), rows
}

// getRuleParametersColumnsAndRows prepares columns and rows for the rule parameters table.
func getRuleParametersColumnsAndRows(ruleDetails rule, setParameters indexedSetParameters) ([]table.Column, []table.Row) {
	var rows []table.Row

	// Sort parameters by ID for consistent ordering in the table
	sort.Slice(ruleDetails.Parameters, func(i, j int) bool {
		return ruleDetails.Parameters[i].ID < ruleDetails.Parameters[j].ID
	})

	for _, param := range ruleDetails.Parameters {
		paramValue := "N/A"
		if values, ok := setParameters[param.ID]; ok && len(values) > 0 {
			paramValue = strings.Join(values, ", ")
		}
		rows = append(rows, table.Row{param.ID, param.Description, paramValue})
	}

	columns := []table.Column{
		{Title: "Parameter ID", Width: 56},
		{Title: "Parameter Description", Width: 56},
		{Title: "Set Value(s)", Width: 30},
	}

	return calculateDynamicColumnWidths(columns, rows), rows
}

func getControlListColumnsAndRows(controls []control) ([]table.Column, []table.Row) {
	// Sort controls by ID for logical ordering in the table
	sort.Slice(controls, func(i, j int) bool {
		return controls[i].ID < controls[j].ID
	})

	var rows []table.Row
	for _, control := range controls {
		var plugins []string
		for _, rule := range control.Rules {
			plugins = append(plugins, rule.Plugin)
		}

		row := table.Row{
			control.ID,
			control.Title,
			control.ImplementationStatus,
			strings.Join(removeDuplicates(plugins), ", "),
		}
		rows = append(rows, row)
	}

	columns := []table.Column{
		{Title: "Control ID", Width: colWidthControlID},
		{Title: "Control Title", Width: colWidthControlTitle},
		{Title: "Status", Width: colWidthImplStatus},
		{Title: "Plugins Used", Width: colWidthPluginsUsed},
	}

	return calculateDynamicColumnWidths(columns, rows), rows
}

// newControlInfoModel creates a Tea model for displaying specific control details.
func newControlInfoModel(control control, rowLimit int) terminal.Model {
	// Prepare the header message with control details
	wrappedDescription := terminal.WrapText(control.Description, 60)
	headerFields := strings.Join([]string{
		renderKeyValuePair("Control ID", control.ID),
		renderKeyValuePair("Title", control.Title),
		renderKeyValuePair("Status", control.ImplementationStatus),
		keyStyle.Render("Description") + ":\n" + valueStyle.Render(wrappedDescription),
	}, "\n")

	finalHeaderOutput := infoContainerStyle.Render(headerFields)

	columns, rows := getControlRulesColumnsAndRows(control)

	tableHeight := calculateRowLimit(rowLimit, len(rows))

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
	)

	tbl.SetStyles(table.Styles{
		Header: tableHeaderStyle,
		Cell:   tableCellStyle,
	})

	return terminal.Model{
		Table:     tbl,
		HeaderMsg: finalHeaderOutput,
		HelpMsg:   fmt.Sprintf("Showing %d of %d available rules. Use --limit to limit table rows.", tableHeight-1, len(rows)),
	}
}

// newRuleInfoModel creates a Bubble Tea model for displaying specific rule details.
func newRuleInfoModel(ruleDetails rule, setParameters indexedSetParameters, rowLimit int) terminal.Model {

	headerFields := strings.Join([]string{
		renderKeyValuePair("Rule ID", ruleDetails.ID),
		renderKeyValuePair("Rule Description", ruleDetails.Description),
	}, "\n")

	finalHeaderOutput := infoContainerStyle.Render(headerFields)

	// Prepare the parameters table
	columns, rows := getRuleParametersColumnsAndRows(ruleDetails, setParameters)

	tableHeight := calculateRowLimit(rowLimit, len(rows))

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight), // +1 for header row
	)

	tbl.SetStyles(table.Styles{
		Header: tableHeaderStyle,
		Cell:   tableCellStyle,
	})

	helpMsg := fmt.Sprintf("Showing %d of %d available parameters. Use --limit to limit table rows.", tableHeight-1, len(rows))
	if len(rows) == 0 {
		helpMsg = "No parameters found for this rule."
	}

	return terminal.Model{
		Table:     tbl,
		HeaderMsg: finalHeaderOutput,
		HelpMsg:   helpMsg,
	}
}

// newParameterInfoModel creates a Bubble Tea model for displaying specific parameter details.
func newParameterInfoModel(parameterID string, paramValues []string, ruleRemarks ruleRemarksMap, remarksProps remarksPropertiesMap, appDir complytime.ApplicationDirectory, validator *validation.SchemaValidator, frameworkID string, rowLimit int) terminal.Model {
	// Extract parameter description from properties
	var paramDescription string

	// Find description for this parameter across all remarks
	for _, props := range remarksProps {
		for _, prop := range props {
			// Check for Parameter_Id patterns
			if isParameterIdProperty(prop.Name) && prop.Value == parameterID {
				// Found the parameter, look for its description in the same property set
				for _, descProp := range props {
					if isParameterDescriptionProperty(descProp.Name) {
						paramDescription = descProp.Value
						break
					}
				}
				break
			}
		}
		if paramDescription != "" {
			break
		}
	}

	paramDetails := parameter{
		ID:          parameterID,
		Description: paramDescription,
		Values:      paramValues,
	}

	usedByRules := findRulesUsingParameter(parameterID, ruleRemarks, remarksProps)

	// Create enhanced header with rule information
	var rulesSummary string
	if len(usedByRules) > 0 {
		if len(usedByRules) <= 5 {
			// Show all rules as a bulleted list
			var rulesList []string
			for _, rule := range usedByRules {
				rulesList = append(rulesList, "• "+rule)
			}
			rulesSummary = "\n" + strings.Join(rulesList, "\n")
		} else {
			// Show first 5 rules as a bulleted list + count
			var rulesList []string
			for i := 0; i < 5; i++ {
				rulesList = append(rulesList, "• "+usedByRules[i])
			}
			rulesSummary = fmt.Sprintf("\n%s\n• ... (%d more)", strings.Join(rulesList, "\n"), len(usedByRules)-5)
		}
	} else {
		rulesSummary = "None"
	}

	headerFields := strings.Join([]string{
		renderKeyValuePair("Parameter ID", paramDetails.ID),
		renderKeyValuePair("Description", paramDetails.Description),
		renderKeyValuePair("Used by Rules", rulesSummary),
	}, "\n")

	finalHeaderOutput := infoContainerStyle.Render(headerFields)

	// Organize the parameter alternatives table
	columns, rows := getParameterDetailsColumnsAndRows(paramDetails, paramValues, ruleRemarks, remarksProps, appDir, validator, frameworkID)

	tableHeight := calculateRowLimit(rowLimit, len(rows))

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
	)

	tbl.SetStyles(table.Styles{
		Header: tableHeaderStyle,
		Cell:   tableCellStyle,
	})

	helpMsg := fmt.Sprintf("Parameter alternatives shown below. Used by %d rule(s). Use --limit to limit table rows.", len(usedByRules))
	if len(rows) == 0 {
		helpMsg = "No alternatives found for this parameter."
	}

	return terminal.Model{
		Table:     tbl,
		HeaderMsg: finalHeaderOutput,
		HelpMsg:   helpMsg,
	}
}

// newControlListModel creates a Bubble Tea model for displaying the list of controls.
func newControlListModel(controls []control, rowLimit int) terminal.Model {
	columns, rows := getControlListColumnsAndRows(controls)

	tableHeight := calculateRowLimit(rowLimit, len(rows))

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
	)

	tbl.SetStyles(table.Styles{
		Header: tableHeaderStyle,
		Cell:   tableCellStyle,
	})

	return terminal.Model{
		Table:   tbl,
		HelpMsg: fmt.Sprintf("Showing %d of %d available controls. Use --limit to limit table rows.", tableHeight-1, len(rows)),
	}
}

// displayControlInfo handles displaying information for a specific control.
func displayControlInfo(opts *infoOptions, controlMap indexedControls) error {
	control, ok := controlMap[opts.controlID]
	if !ok {
		return fmt.Errorf("control '%s' does not exist in workspace", opts.controlID)
	}

	if opts.plain {
		cols, rows := getControlRulesColumnsAndRows(control)

		_, _ = fmt.Fprintf(opts.Out, "Control ID: %s \n", control.ID)
		_, _ = fmt.Fprintf(opts.Out, "Title: %s \n", control.Title)
		_, _ = fmt.Fprintf(opts.Out, "Status: %s \n", control.ImplementationStatus)
		_, _ = fmt.Fprintf(opts.Out, "Description: %s \n", control.Description)
		_, _ = fmt.Fprintln(opts.Out)
		terminal.ShowPlainTable(opts.Out, cols, rows)
		return nil
	} else {
		model := newControlInfoModel(control, opts.limit)
		return runBubbleTeaProgram(model, opts.Out)
	}

}

// displayParameterInfo handles displaying information for a specific parameter.
func displayParameterInfo(opts *infoOptions, parameterID string, indexedSetParameters indexedSetParameters, ruleRemarks ruleRemarksMap, remarksProps remarksPropertiesMap, appDir complytime.ApplicationDirectory, validator *validation.SchemaValidator) error {
	paramValues, ok := indexedSetParameters[parameterID]
	if !ok {
		return fmt.Errorf("parameter '%s' does not exist in framework '%s'", parameterID, opts.complyTimeOpts.FrameworkID)
	}

	var paramDescription string

	// Find description for parameter across all remarks
	for _, props := range remarksProps {
		for _, prop := range props {
			// Check for Parameter_Id patterns
			if isParameterIdProperty(prop.Name) && prop.Value == parameterID {
				// Found the parameter, look for its description in the same property set
				for _, descProp := range props {
					if isParameterDescriptionProperty(descProp.Name) {
						paramDescription = descProp.Value
						break
					}
				}
				break
			}
		}
		if paramDescription != "" {
			break
		}
	}

	paramDetails := parameter{
		ID:          parameterID,
		Description: paramDescription,
		Values:      paramValues,
	}

	if opts.plain {
		_, _ = fmt.Fprintf(opts.Out, "Parameter Information:\n")
		_, _ = fmt.Fprintf(opts.Out, "  ID: %s\n", paramDetails.ID)
		if paramDetails.Description != "" {
			_, _ = fmt.Fprintf(opts.Out, "  Description: %s\n", paramDetails.Description)
		}

		// Get current value and usage
		currentValue := getParameterCurrentValue(paramValues)
		usedByRules := findRulesUsingParameter(paramDetails.ID, ruleRemarks, remarksProps)
		availableAlternatives := findParameterAlternativesFromRemarks(paramDetails.ID, remarksProps)

		_, _ = fmt.Fprintf(opts.Out, "  Current Value: %s\n", currentValue)
		_, _ = fmt.Fprintf(opts.Out, "  Used by %d rule(s)\n", len(usedByRules))
		_, _ = fmt.Fprintf(opts.Out, "  Available Alternatives: %d\n", len(availableAlternatives))
		_, _ = fmt.Fprintln(opts.Out)

		// Show the table of available alternatives
		_, _ = fmt.Fprintf(opts.Out, "Available Alternatives:\n")
		cols, rows := getParameterDetailsColumnsAndRows(paramDetails, paramValues, ruleRemarks, remarksProps, appDir, validator, opts.complyTimeOpts.FrameworkID)
		terminal.ShowPlainTable(opts.Out, cols, rows)
		_, _ = fmt.Fprintln(opts.Out)

		// Show the rules table
		if len(usedByRules) > 0 {
			_, _ = fmt.Fprintf(opts.Out, "Rules Using This Parameter:\n")
			cols, rows = getParameterRulesColumnsAndRows(parameterID, ruleRemarks, remarksProps)
			terminal.ShowPlainTable(opts.Out, cols, rows)
		}
		return nil
	} else {
		model := newParameterInfoModel(parameterID, paramValues, ruleRemarks, remarksProps, appDir, validator, opts.complyTimeOpts.FrameworkID, opts.limit)
		return runBubbleTeaProgram(model, opts.Out)
	}
}

// displayRuleInfo handles displaying information for a specific rule.
func displayRuleInfo(opts *infoOptions, ruleID string, ruleRemarksMap ruleRemarksMap, remarksPropsMap remarksPropertiesMap, setParameters indexedSetParameters) error {
	remarksForRule, ok := ruleRemarksMap[ruleID]
	if !ok || remarksForRule == "" {
		return fmt.Errorf("rule '%s' remarks not found", ruleID)
	}

	propsForRule, ok := remarksPropsMap[remarksForRule]
	if !ok || len(propsForRule) == 0 {
		return fmt.Errorf("properties for rule '%s' (remarks '%s') not found", ruleID, remarksForRule)
	}

	ruleDetails := extractRuleDetails(propsForRule)
	ruleDetails.ID = ruleID // Ensure ID is set for consistency

	if opts.plain {
		_, _ = fmt.Fprintf(opts.Out, "Rule ID: %s \n", ruleDetails.ID)
		_, _ = fmt.Fprintf(opts.Out, "Rule Description: %s \n", ruleDetails.Description)
		_, _ = fmt.Fprintln(opts.Out)
		cols, rows := getRuleParametersColumnsAndRows(ruleDetails, setParameters)
		terminal.ShowPlainTable(opts.Out, cols, rows)
		return nil
	} else {
		model := newRuleInfoModel(ruleDetails, setParameters, opts.limit)
		return runBubbleTeaProgram(model, opts.Out)
	}

}

// displayAllControls handles displaying a list controls in the framework.
func displayAllControls(opts *infoOptions, indexedControls indexedControls) error {
	var controls []control
	for _, control := range indexedControls {
		controls = append(controls, control)
	}

	if opts.plain {
		cols, rows := getControlListColumnsAndRows(controls)
		terminal.ShowPlainTable(opts.Out, cols, rows)
		return nil
	} else {
		model := newControlListModel(controls, opts.limit)
		return runBubbleTeaProgram(model, opts.Out)
	}
}

// calculateRowLimit determines how many rows should be displayed based
// on the number of rows available and the limit set by the user.
func calculateRowLimit(rowLimit int, availableRows int) int {

	if rowLimit > 0 {
		if rowLimit > availableRows {
			return availableRows + 1 // +1 for header row
		} else {
			return rowLimit + 1
		}
	} else {
		return availableRows + 1
	}
}

// getParameterCurrentValue extracts the current value from parameter values (only one selection allowed).
func getParameterCurrentValue(paramValues []string) string {
	if len(paramValues) > 0 {
		return paramValues[0]
	}
	return ""
}

// parseParameterAlternatives parses the Parameter_Value_Alternatives value to extract choice options.
func parseParameterAlternatives(alternativesValue string) []string {
	var alternatives []string

	// Remove outer quotes and braces
	cleaned := strings.Trim(alternativesValue, "\"'")
	cleaned = strings.Trim(cleaned, "{}")

	if cleaned == "" {
		return alternatives
	}

	// Split by comma and extract values
	pairs := strings.Split(cleaned, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) == 2 {
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			if value != "" {
				alternatives = append(alternatives, value)
			}
		}
	}
	// Remove duplicates
	return removeDuplicates(alternatives)
}

// calculateDynamicColumnWidths adjusts column widths based on content for better table display.
func calculateDynamicColumnWidths(columns []table.Column, rows []table.Row) []table.Column {
	for i := range columns {
		maxLength := columns[i].Width
		for _, row := range rows {
			if i < len(row) {
				cellLength := lipgloss.Width(row[i])
				if cellLength > maxLength {
					maxLength = cellLength
				}
			}
		}
		columns[i].Width = maxLength
	}
	return columns
}
