package xccdf

// Goal: identify rule that are available in datastream
// Step 1: Using the access to the profile ID, find the rules
// Step 2: Collect rules as a list
// Step 3: Print list and store and then when updating use a pointer
// https://github.com/ComplianceAsCode/compliance-operator/blob/fed54b4b761374578016d79d97bcb7636bf9d920/pkg/utils/parse_arf_result.go#L170
// Eventually include this file in the datastream.go and reference in tailoring.go for check of rules
import (
	"fmt"
	"github.com/antchfx/xmlquery"
)

// From the compliance-operator parsing results for hash table
// Want mapping of has table to a list
// Need to get length of items in the map
// Use range keyword to iterate through and find the mappings
// Thinking map of slices to append a list

// From tailoring.go but using prefix EX
func ExRuleHashTable(dsDom *xmlquery.Node) NodeByIdHashTable {
	return newHashTableFromRootAndQuery(dsDom, "//ds:component/xccdf-1.2:Benchmark", "//xccdf-1.2:Rule")
}

// This is where the map type is declared from the compliance operator

type NodeByIdHashTable map[string]*xmlquery.Node
type nodeByIdHashTable map[string]*xmlquery.Node

func newxByIdHashTable(nodes []*xmlquery.Node) NodeByIdHashTable {
	table := make(NodeByIdHashTable)
	for i := range nodes {
		ruleDefinition := nodes[i]
		ruleId := ruleDefinition.SelectAttr("id")

		table[ruleId] = ruleDefinition

	}
	return table
}

// Note: we want the profile data to reveal the rule. Use the profile id to access rules

// Using compliance-operator pkg/xccdf/tailoring.go
// The loaded data stream can return the dsDom which will be the parsed file
// Using the getDsElement

// General thought process outlining the process for ruleID extraction - WIP
func GetDsRuleID(ruleId string) string {
	return fmt.Sprintf("xccdf_org.ssgproject.content_profile_%s", ruleId)
}

// Getting Ds RuleIds from the datastream path
func GetDsRuleIds(ruleIds []string, dsPath string) ([]string, error) {
	dsDom, err := loadDataStream(dsPath)
	if err != nil {
		return "", fmt.Errorf("error loading datastream %s", err)
	}
	// Getting each ruleID to format based on id
	// The rules are an element of the profiles, so based on profile id, get rule element
	// Make the rule elements into a list
	//dsRuleID := GetDsRuleID(ruleIds[])
	for ruleID := range ruleIds {
		dsRuleID := GetDsRuleID(ruleIds[])
		//dsRuleID := GetDsRuleID(ruleID[ruleIds])
		rule, err := getDsElement(dsDom, fmt.Sprintf("//xccdf-1.2:Rule[@id= '%s']", dsRuleID))
		if err != nil {
			return "", fmt.Errorf("error processing rule %s in datastream: %s ", dsRuleID, err)
		}
		if rule == nil {
			return "", fmt.Errorf("rule not found: %s", dsRuleID)
		}
	}
	dsRuleID := GetDsRuleID(ruleIds[])
	rule, err := getDsElement(dsDom, fmt.Sprintf("//xccdf-1.2:Rule[@id= '%s']", dsRuleID))
	if err != nil {
		return "", fmt.Errorf("error processing rule %s in datastream: %s ", dsRuleID, err)
	}

	if rule == nil {
		return "", fmt.Errorf("rule not found: %s", dsRuleID)
	}

	ruleTitle, err := xmlquery.Query(rule, "//xccdf-1.2:title")
	if err != nil || ruleTitle == nil {
		return "", fmt.Error("error finding rule title %s: %s", dsRuleID, err)
	}
	return ruleTitle.InnerText(), nil

}
