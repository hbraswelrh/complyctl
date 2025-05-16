// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/complytime/complytime/cmd/complytime/option"
	"github.com/complytime/complytime/internal/complytime"
)

// Config is used for marshalling a go struct into json data.
type Config struct {
	FrameworkID     string   `json:"assessment_plan"`
	Components      string   `json:"components"`
	IncludeControls []string `json:"include_controls"`
	ControlIds      []string `json:"control_ids"`
}

// Configuration formats assessment-plan data as go struct.
type Configuration struct {
	FrameworkID     string   `yaml:"assessment_plan"`
	Components      string   `yaml:"components"`
	IncludeControls []string `yaml:"include_controls"`
	ControlIds      []string `yaml:"control_ids"`
}

// PlanData sets up the yaml mapping type for writing to config file.
// Formats testdata as go struct.
type PlanData struct {
	FrameworkID     string   `yaml:"assessment_plan"`
	Components      []string `yaml:"components"`
	IncludeControls []string `yaml:"include_controls"`
	ControlIds      []string `yaml:"control_ids"`
}

// updateOptions defines options for the "update" subcommand
type updateOptions struct {
	*option.Common
	complyTimeOpts *option.ComplyTime
}

// updateCmd creates a new cobra.Command for the "update" subcommand
func updateCmd(common *option.Common) *cobra.Command {
	updateOpts := &updateOptions{
		Common:         common,
		complyTimeOpts: &option.ComplyTime{},
	}
	cmd := &cobra.Command{
		Use:     "update [flags] id",
		Short:   "Generate a new assessment update for a given compliance framework id.",
		Example: "complytime update myframework",
		Args:    cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				updateOpts.complyTimeOpts.FrameworkID = filepath.Clean(args[0])
			}
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdate(cmd, updateOpts)
		},
	}
	updateOpts.complyTimeOpts.BindFlags(cmd.Flags())
	return cmd
}

func runUpdate(cmd *cobra.Command, opts *updateOptions) error {
	// Create the application directory if it does not exist
	appDir, err := complytime.NewApplicationDirectory(true)
	if err != nil {
		return err
	}
	logger.Debug(fmt.Sprintf("Using application directory: %s", appDir.AppDir()))
	componentDefs, err := complytime.FindComponentDefinitions(appDir.BundleDir())
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("componentDefs %v\n", componentDefs))

	RelayContent()

	return nil
}

// RelayContent leverages the PlanData structure to populate testdata
// test data is written to the config.yaml.
func RelayContent() {
	ycfg := PlanData{
		FrameworkID:     "anssi_bp28_minimal",
		Components:      []string{"rules", "controls", "parameters"},
		IncludeControls: []string{"R1", "R2"},
		ControlIds:      []string{"R1", "R2", "R3", "R4", "R5"},
	}
	out, err := yaml.Marshal(&ycfg)
	if err != nil {
		fmt.Println("error marshalling yaml content: ", err)
	}
	fmt.Println(string(out))

	file, err := os.Create("config.yaml")
	if err != nil {
		fmt.Println("error, the plan updates couldn't be written: ", err)
	}
	defer file.Close()

	// Writing the YAML Data to config.yaml
	_, err = file.Write(out)
	if err != nil {
		fmt.Println("error writing to config: ", err)

	}
	fmt.Println("The updated plan content was written to config.yaml")
}
