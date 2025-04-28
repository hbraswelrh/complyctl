// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"testing"

	"github.com/oscal-compass/oscal-sdk-go/validation"
	"github.com/stretchr/testify/require"
)

func TestLoadFrameworks(t *testing.T) {
	tests := []struct {
		name           string
		appDir         func() ApplicationDirectory
		wantFrameworks []Framework
		wantErr        error
	}{
		{
			name: "Valid/HappyPath",
			appDir: func() ApplicationDirectory {
				appDir, err := newApplicationDirectory("testdata", false)
				require.NoError(t, err)
				return appDir
			},
			wantFrameworks: []Framework{
				{
					Title:               "Example Profile (low)",
					ID:                  "example",
					SupportedComponents: []string{"My Software"},
				},
			},
		},
		{
			name: "Invalid/NoComponentDefinitions",
			appDir: func() ApplicationDirectory {
				tmpDir := t.TempDir()
				appDir, err := newApplicationDirectory(tmpDir, true)
				require.NoError(t, err)
				return appDir
			},
			wantErr: ErrNoComponentDefinitionsFound,
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			appDir := c.appDir()
			gotFrameworks, err := LoadFrameworks(appDir, validation.NoopValidator{})
			if c.wantErr != nil {
				require.ErrorIs(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, c.wantFrameworks, gotFrameworks)
			}
		})
	}
}

func TestLoadProfile(t *testing.T) {
	tests := []struct {
		name    string
		appDir  func() ApplicationDirectory
		source  string
		wantErr string
	}{
		{
			name: "Valid Profile Load",
			appDir: func() ApplicationDirectory {
				appDir, err := newApplicationDirectory("testdata", false)
				require.NoError(t, err)
				return appDir
			},
			source:  "file://controls/sample-profile.json",
			wantErr: "",
		},
		{
			name: "File Does Not Exist",
			appDir: func() ApplicationDirectory {
				appDir, err := newApplicationDirectory("testdata", false)
				require.NoError(t, err)
				return appDir
			},
			source:  "file://nonexistent/path/profile.json",
			wantErr: "got path nonexistent/path/profile.json, control source is expected to be under path testdata/complytime/controls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadProfile(tt.appDir(), tt.source, validation.NoopValidator{})
			if tt.wantErr != "" {
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
