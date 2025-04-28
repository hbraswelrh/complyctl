// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPluginOptions(t *testing.T) {
	tests := []struct {
		name       string
		selections PluginOptions
		wantMap    map[string]string
		wantErr    string
	}{
		{
			name: "Valid/Selections",
			selections: PluginOptions{
				Workspace: "testworkspace",
				Profile:   "testprofile",
			},
			wantMap: map[string]string{
				"workspace": "testworkspace",
				"profile":   "testprofile",
			},
		},
		{
			name:       "Invalid/MissingOptions",
			selections: PluginOptions{},
			wantErr:    "workspace must be set",
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			err := c.selections.Validate()
			if c.wantErr != "" {
				require.EqualError(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
				gotMap := c.selections.ToMap()
				require.Equal(t, c.wantMap, gotMap)
			}
		})
	}
}
