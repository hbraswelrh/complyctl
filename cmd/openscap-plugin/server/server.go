// SPDX-License-Identifier: Apache-2.0
// The code here is still under development and minimally functional for testing purposes.

package server

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/utils"
	"github.com/antchfx/xmlquery"
	"github.com/hashicorp/go-hclog"
	"github.com/oscal-compass/compliance-to-policy-go/v2/policy"

	"github.com/complytime/complytime/cmd/openscap-plugin/config"
	"github.com/complytime/complytime/cmd/openscap-plugin/scan"
	"github.com/complytime/complytime/cmd/openscap-plugin/xccdf"
)

var (
	_ policy.Provider = (*PluginServer)(nil)
	// ovalRegex is a regular expression for capturing the check short name
	// in an OVAL check definition identifier.
	ovalRegex = regexp.MustCompile(`^[^:]*?:[^-]*?-(.*?):.*?$`)
)

const ovalCheckType = "http://oval.mitre.org/XMLSchema/oval-definitions-5"

type PluginServer struct {
	Config *config.Config
}

func New() PluginServer {
	return PluginServer{
		Config: config.NewConfig(),
	}
}

func (s PluginServer) Configure(configMap map[string]string) error {
	return s.Config.LoadSettings(configMap)
}

func (s PluginServer) Generate(policy policy.Policy) error {
	hclog.Default().Info("Generating a tailoring file")
	tailoringXML, err := xccdf.PolicyToXML(policy, s.Config)
	if err != nil {
		return err
	}

	policyPath := s.Config.Files.Policy
	dst, err := os.Create(policyPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := dst.WriteString(tailoringXML); err != nil {
		return err
	}
	return nil
}

func (s PluginServer) GetResults(oscalPolicy policy.Policy) (policy.PVPResult, error) {
	pvpResults := policy.PVPResult{}
	policyChecks := newChecks()

	_, err := scan.ScanSystem(s.Config, s.Config.Parameters.Profile)
	if err != nil {
		return policy.PVPResult{}, err
	}

	policyChecks.LoadPolicy(oscalPolicy)

	// get some results here
	file, err := os.Open(filepath.Clean(s.Config.Files.ARF))
	if err != nil {
		return policy.PVPResult{}, err
	}
	defer file.Close()

	xmlnode, err := utils.ParseContent(bufio.NewReader(file))
	if err != nil {
		return policy.PVPResult{}, err
	}

	ruleTable := xccdf.NewRuleHashTable(xmlnode)
	results := xmlnode.SelectElements("//rule-result")
	for i := range results {
		result := results[i]
		ruleIDRef := result.SelectAttr("idref")

		rule, ok := ruleTable[ruleIDRef]
		if !ok {
			continue
		}

		var ovalRefEl *xmlquery.Node
		for _, check := range rule.SelectElements("//xccdf-1.2:check") {
			if check.SelectAttr("system") == ovalCheckType {
				ovalRefEl = check.SelectElement("xccdf-1.2:check-content-ref")
				break
			}
		}
		if ovalRefEl == nil {
			continue
		}
		ovalCheck, err := parseCheck(ovalRefEl)
		if err != nil {
			return policy.PVPResult{}, err
		}
		if policyChecks.Has(ovalCheck) {
			mappedResult, err := mapResultStatus(result)
			if err != nil {
				return policy.PVPResult{}, err
			}
			observation := policy.ObservationByCheck{
				Title:     ruleIDRef,
				Methods:   []string{"AUTOMATED"},
				Collected: time.Now(),
				CheckID:   ovalCheck,
				Subjects: []policy.Subject{
					{
						Title:       "My Comp",
						Type:        "component",
						ResourceID:  ruleIDRef,
						EvaluatedOn: time.Now(),
						Result:      mappedResult,
						Reason:      "my reason",
					},
				},
				RelevantEvidences: []policy.Link{
					{
						Href:        fmt.Sprintf("file://%s", s.Config.Files.ARF),
						Description: "ARF_FILE",
					},
				},
			}
			pvpResults.ObservationsByCheck = append(pvpResults.ObservationsByCheck, observation)
		}
	}
	return pvpResults, nil
}

// checks is a Set implementation for comparing OSCAL
// and OVAL checks ids.
type checks map[string]struct{}

func newChecks() checks {
	policyChecks := make(checks)
	return policyChecks
}

func (c checks) LoadPolicy(oscalPolicy policy.Policy) {
	for _, rule := range oscalPolicy {
		for _, check := range rule.Checks {
			c[check.ID] = struct{}{}
		}
	}
}

func (c checks) Has(check string) bool {
	_, ok := c[check]
	return ok
}

// parseCheck returns the check short name without the OVAL-specific naming from a
// rule in results.
func parseCheck(check *xmlquery.Node) (string, error) {
	ovalCheckName := strings.TrimSpace(check.SelectAttr("name"))
	if ovalCheckName == "" {
		return "", errors.New("check-content-ref node has no 'name' attribute")
	}
	matches := ovalRegex.FindStringSubmatch(ovalCheckName)

	minimumPart, shortNameLoc := 2, 1
	if len(matches) < minimumPart {
		return "", fmt.Errorf("check id %q is in unexpected format", ovalCheckName)
	}
	trimmedCheckName := matches[shortNameLoc]
	return trimmedCheckName, nil
}

func mapResultStatus(result *xmlquery.Node) (policy.Result, error) {
	resultEl := result.SelectElement("result")
	if resultEl == nil {
		return policy.ResultInvalid, errors.New("result node has no 'result' attribute")
	}
	switch resultEl.InnerText() {
	case "pass", "fixed":
		return policy.ResultPass, nil
	case "fail":
		return policy.ResultFail, nil
	case "notselected", "notapplicable":
		return policy.ResultError, nil
	case "error", "unknown":
		return policy.ResultError, nil
	}

	return policy.ResultInvalid, fmt.Errorf("couldn't match %s", resultEl.InnerText())
}
