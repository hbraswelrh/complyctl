// SPDX-License-Identifier: Apache-2.0

package xccdf

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/xccdf"
	"github.com/oscal-compass/compliance-to-policy-go/v2/policy"

	"github.com/complytime/complytime/cmd/openscap-plugin/config"
)

const (
	XCCDFCaCNamespace    string = "xccdf_org.ssgproject.content"
	XCCDFNamespace       string = "complytime.openscapplugin"
	XCCDFTailoringSuffix string = "complytime"
)

func removePrefix(str, prefix string) string {
	return strings.TrimPrefix(str, prefix)
}

func getTailoringID() string {
	return fmt.Sprintf("xccdf_%s_tailoring_%s", XCCDFNamespace, XCCDFTailoringSuffix)
}

func getTailoringExtendedProfileID(profileId string) string {
	return fmt.Sprintf(
		"%s_profile_%s", XCCDFCaCNamespace, profileId)
}

func getTailoringProfileID(profileId string) string {
	return fmt.Sprintf(
		"xccdf_%s_profile_%s_%s", XCCDFNamespace, profileId, XCCDFTailoringSuffix)
}

func getTailoringProfileTitle(dsProfileTitle string) string {
	return fmt.Sprintf("ComplyTime Tailoring Profile - %s", dsProfileTitle)
}

func getTailoringVersion() xccdf.VersionElement {
	return xccdf.VersionElement{
		Time:  time.Now().Format(time.RFC3339),
		Value: "1",
	}
}

func getTailoringBenchmarkHref(datastreamPath string) xccdf.BenchmarkElement {
	return xccdf.BenchmarkElement{
		Href: datastreamPath,
	}
}

func validateRuleExistence(policyRuleID string, dsRules []DsRules) bool {
	for _, dsRule := range dsRules {
		ruleID := removePrefix(dsRule.ID, ruleIDPrefix)
		if policyRuleID == ruleID {
			return true
		}
	}
	return false
}

func validateVariableExistence(policyVariableID string, dsVariables []DsVariables) bool {
	for _, dsVariable := range dsVariables {
		varID := removePrefix(dsVariable.ID, varIDPrefix)
		if policyVariableID == varID {
			return true
		}
	}
	return false
}

func unselectAbsentRules(tailoringSelections, dsProfileSelections []xccdf.SelectElement, oscalPolicy policy.Policy) []xccdf.SelectElement {
	for _, dsRule := range dsProfileSelections {
		dsRuleAlsoInPolicy := false
		ruleID := removePrefix(dsRule.IDRef, ruleIDPrefix)
		for _, rule := range oscalPolicy {
			if ruleID == rule.Rule.ID {
				dsRuleAlsoInPolicy = true
				break
			}
		}
		if !dsRuleAlsoInPolicy && dsRule.Selected {
			tailoringSelections = append(tailoringSelections, xccdf.SelectElement{
				IDRef:    dsRule.IDRef,
				Selected: false,
			})
		}
	}
	return tailoringSelections
}

func selectAdditionalRules(tailoringSelections, dsProfileSelections []xccdf.SelectElement, oscalPolicy policy.Policy) []xccdf.SelectElement {
	rulesMap := make(map[string]bool)

	for _, rule := range oscalPolicy {
		ruleAlreadyInDsProfile := false
		for _, dsRule := range dsProfileSelections {
			dsRuleID := removePrefix(dsRule.IDRef, ruleIDPrefix)
			if rule.Rule.ID == dsRuleID {
				// Not a common case, but a rule can be unselected in a Datastream Profile
				if dsRule.Selected {
					ruleAlreadyInDsProfile = true
				}
				break
			}
		}
		ruleID := getDsRuleID(rule.Rule.ID)
		if !ruleAlreadyInDsProfile && !rulesMap[ruleID] {
			rulesMap[ruleID] = true
			tailoringSelections = append(tailoringSelections, xccdf.SelectElement{
				IDRef:    ruleID,
				Selected: true,
			})
		}
	}
	return tailoringSelections
}

func getTailoringSelections(oscalPolicy policy.Policy, dsProfile *xccdf.ProfileElement, dsPath string) ([]xccdf.SelectElement, error) {
	dsRules, err := GetDsRules(dsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules from datastream: %w", err)
	}

	// All OSCAL Policy rules should be present in the Datastream
	for _, rule := range oscalPolicy {
		if !validateRuleExistence(rule.Rule.ID, dsRules) {
			return nil, fmt.Errorf("rule %s not found in datastream: %s", rule.Rule.ID, dsPath)
		}
	}

	var tailoringSelections []xccdf.SelectElement
	// Rules in dsProfile but not in OSCAL Policy must be unselected in Tailoring file.
	tailoringSelections = unselectAbsentRules(tailoringSelections, dsProfile.Selections, oscalPolicy)
	tailoringSelections = selectAdditionalRules(tailoringSelections, dsProfile.Selections, oscalPolicy)

	return tailoringSelections, nil
}

func updateTailoringValues(tailoringValues, dsProfileValues []xccdf.SetValueElement, oscalPolicy policy.Policy) []xccdf.SetValueElement {
	varsMap := make(map[string]bool)

	for _, rule := range oscalPolicy {
		if rule.Rule.Parameter == nil {
			continue
		}
		varAlreadyInDsProfile := false
		for _, dsVar := range dsProfileValues {
			dsVarID := removePrefix(dsVar.IDRef, varIDPrefix)
			if rule.Rule.Parameter.ID == dsVarID {
				if rule.Rule.Parameter.Value == dsVar.Value {
					varAlreadyInDsProfile = true
				}
				break
			}
		}
		varID := getDsVarID(rule.Rule.Parameter.ID)
		if !varAlreadyInDsProfile && !varsMap[varID] {
			varsMap[varID] = true
			tailoringValues = append(tailoringValues, xccdf.SetValueElement{
				IDRef: varID,
				Value: rule.Rule.Parameter.Value,
			})
		}
	}
	return tailoringValues
}

func getTailoringValues(oscalPolicy policy.Policy, dsProfile *xccdf.ProfileElement, dsPath string) ([]xccdf.SetValueElement, error) {
	dsVariables, err := GetDsVariablesValues(dsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get variables from datastream: %w", err)
	}

	// All OSCAL policy variables should be present in the Datastream
	for _, rule := range oscalPolicy {
		if rule.Rule.Parameter == nil {
			continue
		}
		if !validateVariableExistence(rule.Rule.Parameter.ID, dsVariables) {
			return nil, fmt.Errorf("variable %s not found in datastream: %s", rule.Rule.Parameter.ID, dsPath)
		}
	}

	dsProfile, err = ResolveDsVariableOptions(dsProfile, dsVariables)
	if err != nil {
		return nil, fmt.Errorf("failed to get values from variables options: %w", err)
	}

	var tailoringValues []xccdf.SetValueElement
	tailoringValues = updateTailoringValues(tailoringValues, dsProfile.Values, oscalPolicy)

	return tailoringValues, nil
}

func getTailoringProfile(profileId string, dsPath string, oscalPolicy policy.Policy) (*xccdf.ProfileElement, error) {
	tailoringProfile := new(xccdf.ProfileElement)
	tailoringProfile.ID = getTailoringProfileID(profileId)

	dsProfile, err := GetDsProfile(profileId, dsPath)
	if err != nil {
		return tailoringProfile, fmt.Errorf("failed to get base profile from datastream: %w", err)
	}

	tailoringProfile.Extends = getTailoringExtendedProfileID(profileId)

	tailoringProfile.Title = &xccdf.TitleOrDescriptionElement{
		Override: true,
		Value:    getTailoringProfileTitle(dsProfile.Title.Value),
	}

	tailoringProfile.Selections, err = getTailoringSelections(oscalPolicy, dsProfile, dsPath)
	if err != nil {
		return tailoringProfile, fmt.Errorf("failed to get selections for tailoring profile: %w", err)
	}

	tailoringProfile.Values, err = getTailoringValues(oscalPolicy, dsProfile, dsPath)
	if err != nil {
		return tailoringProfile, fmt.Errorf("failed to get values for tailoring profile: %w", err)
	}
	return tailoringProfile, nil
}

func PolicyToXML(oscalPolicy policy.Policy, config *config.Config) (string, error) {
	datastreamPath := config.Files.Datastream
	profileId := config.Parameters.Profile

	if oscalPolicy == nil {
		return "", fmt.Errorf("OSCAL policy is empty")
	}

	tailoringProfile, err := getTailoringProfile(profileId, datastreamPath, oscalPolicy)
	if err != nil {
		return "", err
	}

	tailoring := xccdf.TailoringElement{
		XMLNamespaceURI: xccdf.XCCDFURI,
		ID:              getTailoringID(),
		Version:         getTailoringVersion(),
		Benchmark:       getTailoringBenchmarkHref(datastreamPath),
		Profile:         *tailoringProfile,
	}

	output, err := xml.MarshalIndent(tailoring, "", "  ")
	if err != nil {
		return "", err
	}
	return xccdf.XMLHeader + "\n" + string(output), nil
}
