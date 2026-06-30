// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/complytime/complyctl/internal/cache"
)

func TestPolicyState_BackwardCompatibility_NoVerifiedField(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a state.json without the "verified" field (simulating a
	// file from a previous version of complyctl).
	legacyState := map[string]interface{}{
		"last_sync": "2025-01-01T00:00:00Z",
		"policies": map[string]interface{}{
			"legacy-policy": map[string]interface{}{
				"version":      "v1.0.0",
				"digest":       "sha256:legacy",
				"last_updated": "2025-01-01T00:00:00Z",
			},
		},
	}
	data, err := json.MarshalIndent(legacyState, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "state.json"), data, 0600)
	require.NoError(t, err)

	loaded, err := cache.LoadState(tmpDir)
	require.NoError(t, err)
	ps, ok := loaded.GetPolicyState("legacy-policy")
	require.True(t, ok)
	assert.Equal(t, "v1.0.0", ps.Version)
	assert.Equal(t, "sha256:legacy", ps.Digest)
}
