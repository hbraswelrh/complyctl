// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"testing"

	"github.com/oscal-compass/oscal-sdk-go/validation"
	"github.com/stretchr/testify/require"
)

func TestLoadCatalogSource(t *testing.T) {
	tests := []struct {
		name    string
		appDir  func() ApplicationDirectory
		source  string
		wantErr string
	}{
		{
			name: "Valid Catalog Load",
			appDir: func() ApplicationDirectory {
				appDir, err := newApplicationDirectory("testdata", false)
				require.NoError(t, err)
				return appDir
			},
			source:  "file://controls/sample-catalog.json",
			wantErr: "",
		},
		{
			name: "File Does Not Exist",
			appDir: func() ApplicationDirectory {
				appDir, err := newApplicationDirectory("testdata", false)
				require.NoError(t, err)
				return appDir
			},
			source:  "file://nonexistent/path/catalog.json",
			wantErr: "got path nonexistent/path/catalog.json, control source is expected to be under path testdata/complytime/controls",
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
