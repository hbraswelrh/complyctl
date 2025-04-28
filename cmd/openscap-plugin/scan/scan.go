// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"fmt"
	"os"

	"github.com/complytime/complytime/cmd/openscap-plugin/config"
	"github.com/complytime/complytime/cmd/openscap-plugin/oscap"
	"github.com/complytime/complytime/cmd/openscap-plugin/xccdf"
)

func validateOpenSCAPFiles(cfg *config.Config) (map[string]string, error) {
	if _, err := os.Stat(cfg.Files.Policy); err != nil {
		return nil, err
	}

	isXML, err := config.IsXMLFile(cfg.Files.Policy)
	if err != nil || !isXML {
		return nil, err
	}

	return map[string]string{
		"datastream": cfg.Files.Datastream,
		"policy":     cfg.Files.Policy,
		"results":    cfg.Files.Results,
		"arf":        cfg.Files.ARF,
	}, nil
}

func ScanSystem(cfg *config.Config, profile string) ([]byte, error) {
	openscapFiles, err := validateOpenSCAPFiles(cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid openscap files: %w", err)
	}

	tailoringProfile := fmt.Sprintf("%s_%s", profile, xccdf.XCCDFTailoringSuffix)
	// In the future, we can add an integrity check to confirm if the expected tailoring profile
	// id exists in the tailoring file. It is not a common case but a guardrail to prevent manual
	// manipulation of the tailoring file would be good.

	output, err := oscap.OscapScan(openscapFiles, tailoringProfile)
	if err != nil {
		return output, fmt.Errorf("failed during scan: %w", err)
	}

	return output, nil
}
