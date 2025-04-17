package cli

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
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

// PlanConfigYAML populates the config.yaml from defaults.
func PlanConfigYAML() {
	y := Configuration{
		FrameworkID:     "complytime",
		Components:      "controls",
		IncludeControls: []string{"R1", "R2", "R3", "R4", "R5"},
		ControlIds:      []string{"R1", "R2", "R3", "R4", "R5"},
	}
	yamlfile, err := os.Create("config.yaml")
	if err != nil {
		fmt.Println(err)
	}
	encodeFile := yaml.NewEncoder(yamlfile)
	err = encodeFile.Encode(y)
	if err != nil {
		fmt.Println("There was an error encoding the plan_config.", err)
	}
	err = yamlfile.Close()
	if err != nil {
		return
	}
	fmt.Println("\nConfiguration successfully written to: ", yamlfile.Name())
	fmt.Println("\nframework_id: ", y.FrameworkID)
	fmt.Println("\ncomponents: ", y.Components)
	fmt.Println("\nincluded_controls: ", y.IncludeControls)
	fmt.Println("\ncontrol_ids: ", y.ControlIds)

}

// PlanConfigJSON populates config.json defaults from data.
func PlanConfigJSON() {

	c := Config{
		FrameworkID:     "anssi_bp28_minimal",
		Components:      "controls",
		IncludeControls: []string{"R1", "R2", "R3", "R4", "R5"},
		ControlIds:      []string{"R1", "R2", "R3", "R4", "R5"},
	}
	filepath, err := os.Create("config.json")
	if err != nil {
		fmt.Println("error creating the config.json for updating:", err)
		panic(err)
	}
	defer func(filepath *os.File) {
		err := filepath.Close()
		if err != nil {
			fmt.Println("error closing the config.json for updating:", err)
			panic(err)
		}
	}(filepath)

	encoder := json.NewEncoder(filepath)
	err = encoder.Encode(c)
	if err != nil {
		fmt.Println("error with encoding the config: ", err)
		panic(err)
	}
	// Printing to stdout, WIP
	fmt.Printf("\nThe encoded file was successfully written to: %v", filepath.Name())
	fmt.Println("\nframework_id:", c.FrameworkID)
	fmt.Println("\ncomponents:", c.Components)
	fmt.Println("\ncontrol_ids:", c.ControlIds)
	fmt.Println("\ninclude_controls:", c.IncludeControls)
	fmt.Println("\nconfig.json successfully written with updates")
}
