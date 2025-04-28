// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSanitizeInput tests the SanitizeInput function with various valid and invalid inputs.
func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		expectError bool
	}{
		// Valid inputs
		{"valid-input", "valid-input", false},
		{"another_valid.input", "another_valid.input", false},
		{"CAPS_and_numbers123", "CAPS_and_numbers123", false},
		{"mixed-123.UP_case", "mixed-123.UP_case", false},

		// Invalid inputs
		{"invalid/input", "", true},     // contains /
		{"input with spaces", "", true}, // contains spaces
		{"invalid@input", "", true},     // contains @
		{"<invalid>", "", true},         // contains < >
		{";ls", "", true},               // contains ;
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := SanitizeInput(tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
			}
			if result != tt.expected {
				t.Errorf("Expected result: %s, got: %s", tt.expected, result)
			}
		})
	}
}

// TestSanitizePath tests the SanitizePath function with various inputs.
func TestSanitizePath(t *testing.T) {
	usr, _ := user.Current()
	homeDir := usr.HomeDir

	tests := []struct {
		input       string
		expected    string
		expectError bool
	}{
		// Normalizing paths
		{"/foo/bar/../baz", "/foo/baz", false},
		{"./foo/bar", "foo/bar", false},
		{"foo/./bar", "foo/bar", false},
		{"foo/bar/..", "foo", false},
		{"/foo//bar", "/foo/bar", false},
		{"foo//bar//baz", "foo/bar/baz", false},
		{"foo/bar/../../baz", "baz", false},
		{"./../foo", "../foo", false},

		// Expanding paths
		{"~/foo/bar", filepath.Join(homeDir, "foo", "bar"), false},
		{"~", homeDir, false},

		// Weird but valid cases
		{"~weird", "~weird", false}, // not common but possible
		{"", ".", false},            // empty path is updated to the current directory
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := SanitizePath(tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
			}
			if result != tt.expected {
				t.Errorf("Expected result: %s, got: %s", tt.expected, result)
			}
		})
	}
}

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

func TestIsXMLFile(t *testing.T) {
	if err := setupTestFiles(); err != nil {
		t.Fatalf("Failed to setup test files: %v", err)
	}
	defer teardownTestFiles()

	tests := []struct {
		name      string
		filePath  string
		want      bool
		expectErr bool
	}{
		{
			name:      "Valid XML file",
			filePath:  "testdata/valid.xml",
			want:      true,
			expectErr: false,
		},
		{
			name:      "Invalid XML file",
			filePath:  "testdata/invalid.xml",
			want:      false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isXML, err := IsXMLFile(tt.filePath)
			if (err != nil) != tt.expectErr {
				t.Errorf("IsXMLFile(%s) error = %v, expectErr %v", tt.filePath, err, tt.expectErr)
				return
			}
			if isXML != tt.want {
				t.Errorf("IsXMLFile() = %v, want %v", isXML, tt.want)
			}
		})
	}
}

// TestEnsureDirectory tests the ensureDirectory function with various cases.
func TestEnsureDirectory(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		path        string
		expectError bool
	}{
		// Valid cases
		{filepath.Join(tempDir, "absent_dir"), false},   // directory does not exist, should be created
		{filepath.Join(tempDir, "existing_dir"), false}, // directory already exists

		// Invalid cases
		{tempDir + "/invalid\000dir", true}, // invalid directory name
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if tt.path == filepath.Join(tempDir, "existing_dir") {
				// Create directory for existing_dir test
				if err := os.MkdirAll(tt.path, 0750); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
			}

			err := ensureDirectory(tt.path)
			if (err != nil) != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
			}

			// Check if directory was created
			if !tt.expectError {
				if _, err := os.Stat(tt.path); os.IsNotExist(err) {
					t.Errorf("Expected directory to be created: %s", tt.path)
				}
			}
		})
	}
}

// TestEnsureWorkspace tests the ensureWorkspace function with various cases.
func TestEnsureWorkspace(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		cfg         Config
		expectError bool
	}{
		{
			cfg: Config{
				Files: struct {
					Workspace  string "config:\"workspace\""
					Datastream string "config:\"datastream\""
					Results    string "config:\"results\""
					ARF        string "config:\"arf\""
					Policy     string "config:\"policy\""
				}{
					Workspace: filepath.Join(tempDir, "workspace"),
					Policy:    "policy.yaml",
					Results:   "results.xml",
					ARF:       "arf.xml",
				},
			},
			expectError: false,
		},
		{
			cfg: Config{
				Files: struct {
					Workspace  string "config:\"workspace\""
					Datastream string "config:\"datastream\""
					Results    string "config:\"results\""
					ARF        string "config:\"arf\""
					Policy     string "config:\"policy\""
				}{
					Workspace: filepath.Join(tempDir, "invalid\000workspace"),
					Policy:    "policy.yaml",
					Results:   "results.xml",
					ARF:       "arf.xml",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.cfg.Files.Workspace, func(t *testing.T) {
			directories, err := ensureWorkspace(&tt.cfg)
			if (err != nil) != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError {
				for _, dir := range directories {
					if _, err := os.Stat(dir); os.IsNotExist(err) {
						t.Errorf("Expected directory to be created: %s", dir)
					}
				}
			}
		})
	}
}

// TestDefineFilesPaths tests the defineFilesPaths function with various cases.
func TestDefineFilesPaths(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		cfg         Config
		expectError bool
	}{
		{
			cfg: Config{
				Files: struct {
					Workspace  string "config:\"workspace\""
					Datastream string "config:\"datastream\""
					Results    string "config:\"results\""
					ARF        string "config:\"arf\""
					Policy     string "config:\"policy\""
				}{
					Workspace:  filepath.Join(tempDir, "workspace"),
					Datastream: filepath.Join(tempDir, "datastream.xml"),
					Results:    "results.xml",
					ARF:        "arf.xml",
					Policy:     "policy.yaml",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.cfg.Files.Workspace, func(t *testing.T) {
			err := defineFilesPaths(&tt.cfg)
			if (err != nil) != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError {
				// Check if the paths are correctly set
				expectedPolicyPath := filepath.Join(tempDir, "workspace", PluginDir, "policy", "policy.yaml")
				expectedResultsPath := filepath.Join(tempDir, "workspace", PluginDir, "results", "results.xml")
				expectedARFPath := filepath.Join(tempDir, "workspace", PluginDir, "results", "arf.xml")

				if tt.cfg.Files.Policy != expectedPolicyPath {
					t.Errorf("Expected policy path: %s, got: %s", expectedPolicyPath, tt.cfg.Files.Policy)
				}
				if tt.cfg.Files.Results != expectedResultsPath {
					t.Errorf("Expected results path: %s, got: %s", expectedResultsPath, tt.cfg.Files.Results)
				}
				if tt.cfg.Files.ARF != expectedARFPath {
					t.Errorf("Expected ARF path: %s, got: %s", expectedARFPath, tt.cfg.Files.ARF)
				}
			}
		})
	}
}

func TestConfig_LoadSettings(t *testing.T) {
	tempDir := t.TempDir()
	tempDataStream := filepath.Join(tempDir, "datastream.xml")
	err := os.WriteFile(tempDataStream, []byte("example"), 0400)
	require.NoError(t, err)

	tests := []struct {
		name          string
		inputSettings map[string]string
		expectError   string
		wantCfg       Config
	}{
		{
			name: "Valid/AllSettingsSupplied",
			inputSettings: map[string]string{
				"workspace":  tempDir,
				"datastream": tempDataStream,
				"results":    "results.xml",
				"arf":        "arf.xml",
				"policy":     "policy.yaml",
				"profile":    "test",
			},
			wantCfg: Config{
				Files: struct {
					Workspace  string "config:\"workspace\""
					Datastream string "config:\"datastream\""
					Results    string "config:\"results\""
					ARF        string "config:\"arf\""
					Policy     string "config:\"policy\""
				}{
					Workspace:  tempDir,
					Datastream: tempDataStream,
					Results:    filepath.Join(tempDir, "openscap", "results", "results.xml"),
					ARF:        filepath.Join(tempDir, "openscap", "results", "arf.xml"),
					Policy:     filepath.Join(tempDir, "openscap", "policy", "policy.yaml"),
				},
				Parameters: struct {
					Profile string `config:"profile"`
				}{Profile: "test"},
			},
			expectError: "",
		},
		{
			name: "Invalid/MissingSettings",
			inputSettings: map[string]string{
				"workspace":  tempDir,
				"datastream": tempDataStream,
				"results":    "results.xml",
				"arf":        "arf.xml",
				"policy":     "policy.yaml",
			},
			expectError: "missing configuration value for option \"profile\" (field: Profile)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotConfig := NewConfig()
			err := gotConfig.LoadSettings(tt.inputSettings)

			if tt.expectError != "" {
				require.EqualError(t, err, tt.expectError)
			} else {
				require.Equal(t, tt.wantCfg, *gotConfig)
				require.NoError(t, err)
			}
		})
	}
}
