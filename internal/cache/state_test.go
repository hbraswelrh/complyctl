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

func TestPolicyState_VerifiedFieldPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	state := &cache.State{
		Policies: make(map[string]cache.PolicyState),
	}
	state.UpdatePolicyState("test-policy", "v1.0.0", "sha256:abc123", false)

	err := cache.SaveState(state, tmpDir)
	require.NoError(t, err)

	// Read raw JSON to verify "verified" field is explicitly present
	data, err := os.ReadFile(filepath.Join(tmpDir, "state.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"verified": false`,
		"verified field must be explicitly present in serialized state")

	// Round-trip: load and verify the field
	loaded, err := cache.LoadState(tmpDir)
	require.NoError(t, err)
	ps, ok := loaded.GetPolicyState("test-policy")
	require.True(t, ok)
	assert.False(t, ps.Verified, "verified must be false when set to false")
}

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
	assert.False(t, ps.Verified,
		"missing verified field must default to false for backward compatibility")
	assert.Equal(t, "v1.0.0", ps.Version)
	assert.Equal(t, "sha256:legacy", ps.Digest)
}
