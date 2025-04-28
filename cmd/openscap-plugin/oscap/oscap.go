// SPDX-License-Identifier: Apache-2.0

package oscap

import (
	"fmt"
	"os/exec"

	"github.com/hashicorp/go-hclog"
)

func constructScanCommand(openscapFiles map[string]string, profile string) []string {
	datastream := openscapFiles["datastream"]
	tailoringFile := openscapFiles["policy"]
	resultsFile := openscapFiles["results"]
	arfFile := openscapFiles["arf"]

	cmd := []string{
		"oscap",
		"xccdf",
		"eval",
		"--profile", profile,
		"--results", resultsFile,
		"--results-arf", arfFile,
		"--tailoring-file", tailoringFile,
		datastream,
	}

	return cmd
}

func OscapScan(openscapFiles map[string]string, profile string) ([]byte, error) {
	command := constructScanCommand(openscapFiles, profile)

	cmdPath, err := exec.LookPath(command[0])
	if err != nil {
		return nil, fmt.Errorf("command not found: %s", command[0])
	}

	hclog.Default().Info("Executing command", "command", command)
	cmd := exec.Command(cmdPath, command[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if err.Error() == "exit status 1" {
			return output, fmt.Errorf("%s: oscap error during evaluation", err)
		} else if err.Error() == "exit status 2" {
			hclog.Default().Warn("at least one rule resulted in fail or unknown", "err", err)
			return output, nil
		} else {
			hclog.Default().Warn("Error", "err", err)
			return output, nil
		}
	}

	return output, nil
}
