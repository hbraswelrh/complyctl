// SPDX-License-Identifier: Apache-2.0
package scan

import (
	"fmt"
	"os"
	"testing"

	"github.com/complytime/complytime/cmd/openscap-plugin/config"
)

func setupTestFiles() error {
	if err := os.MkdirAll("testdata", os.ModePerm); err != nil {
		return err
	}

	if err := os.WriteFile("testdata/valid.xml", []byte(`<root></root>`), 0600); err != nil {
		return err
	}
	if err := os.WriteFile("testdata/invalid.xml", []byte(`<root>`), 0600); err != nil {
		return err
	}
	return nil
}

func teardownTestFiles() {
	os.RemoveAll("testdata")
}

func TestMain(m *testing.M) {
	if err := setupTestFiles(); err != nil {
		fmt.Printf("Failed to setup test files: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	teardownTestFiles()
	os.Exit(code)
}

// TestValidateOpenSCAPFiles tests the validateOpenSCAPFiles function when a policy file is present,
// after the generate command, or absent before the generate command.
// validateOpenSCAPFiles consumes other functions already tested so the only part to be tested here
// is the policy file existence.
func TestValidateOpenSCAPFiles(t *testing.T) {
	cfg := new(config.Config)
	cfg.Files.Datastream = "testdata/valid.xml"

	tests := []struct {
		name    string
		cfgPol  string
		wantErr bool
	}{
		{
			name:    "present and valid policy file",
			cfgPol:  "testdata/valid.xml",
			wantErr: false,
		},
		{
			name:    "absent policy file",
			cfgPol:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Files.Policy = tt.cfgPol
			_, err := validateOpenSCAPFiles(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOpenSCAPFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

// ScanSystem function is not tested because it is high-level functions using other functions
// already tested above or in other packages.
