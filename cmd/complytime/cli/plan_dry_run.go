// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"fmt"

	"github.com/goccy/go-yaml"

	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
)

// PlanData sets up the yaml mapping type for writing to config file.
// Formats testdata as go struct.
type PlanData struct {
	FrameworkID string   `yaml:"assessment_plan"`
	Controls    []string `yaml:"controls"`
}

// planDryRun leverages the PlanData structure to populate tailoring config.
// The config is written to stdout.
func planDryRun(frameworkId string, cds []oscalTypes.ComponentDefinition) error {
	basePlanData := PlanData{
		FrameworkID: frameworkId,
		Controls:    []string{},
	}
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
			for _, ci := range *component.ControlImplementations {
				if ci.ImplementedRequirements == nil {
					continue
				}
				for _, ir := range ci.ImplementedRequirements {
					if ir.ControlId != "" {
						basePlanData.Controls = append(basePlanData.Controls, ir.ControlId)
					}
				}
			}
		}
	}

	out, err := yaml.Marshal(&basePlanData)
	if err != nil {
		return fmt.Errorf("error marshalling yaml content: %v", err)
	}
	fmt.Println(string(out))
	return nil
}
