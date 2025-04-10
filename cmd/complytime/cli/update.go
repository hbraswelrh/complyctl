package cli

import (
	"encoding/json"
	"fmt"
	"os"

)

// Structure for marshalling a go struct into json data
type Config struct {
	FrameworkID     string   `json:"assessment_plan"`
	Components      string   `json:"components"`
	IncludeControls []string `json:"include_controls"`
	ControlIds      []string `json:"control_ids"`
}
// Function called in main that will populate config.json defaults
func Write() {
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
	fmt.Println("\nframework_id:",c.FrameworkID)
	fmt.Println("\ncomponents:",c.Components)
	fmt.Println("\ncontrol_ids:", c.ControlIds)
	fmt.Println("\ninclude_controls:", c.IncludeControls)
	fmt.Println("\nconfig.json successfully written with updates")
}
