// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/complytime/complyctl/internal/cache"
	"github.com/complytime/complypack/pkg/complypack"
)

// --- ValidatePathComponent tests ---

func TestValidatePathComponent_Valid(t *testing.T) {
	valid := []string{
		"io.complytime.opa",
		"1.0.0",
		"my-pack",
	}
	for _, v := range valid {
		t.Run(v, func(t *testing.T) {
			err := cache.ValidatePathComponent(v)
			assert.NoError(t, err, "expected %q to be accepted", v)
		})
	}
}

func TestValidatePathComponent_PathTraversal(t *testing.T) {
	cases := []string{
		"../../etc",
		"foo/bar",
		`foo\bar`,
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			err := cache.ValidatePathComponent(c)
			require.Error(t, err, "expected %q to be rejected", c)
		})
	}
}

func TestValidatePathComponent_NullByte(t *testing.T) {
	err := cache.ValidatePathComponent("foo\x00bar")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "null bytes")
}

func TestValidatePathComponent_Empty(t *testing.T) {
	err := cache.ValidatePathComponent("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

// --- ComplypackCache tests ---

// newTestConfig returns a complypack.Config suitable for testing.
func newTestConfig(evaluatorID, version string) complypack.Config {
	return complypack.Config{
		EvaluatorID: evaluatorID,
		Version:     version,
	}
}

func TestComplypackCache_Store_CreatesDirectoryStructure(t *testing.T) {
	cacheDir := t.TempDir()
	cc := cache.NewComplypackCache(cacheDir)

	cfg := newTestConfig("io.complytime.opa", "1.0.0")
	contentPath, err := cc.Store(cfg, strings.NewReader("test content"))
	require.NoError(t, err)

	// Verify content.tar.gz exists at the returned path.
	assert.FileExists(t, contentPath)
	assert.True(t, strings.HasSuffix(contentPath, "content.tar.gz"),
		"returned path should end with content.tar.gz, got %s", contentPath)

	// Verify config.json exists alongside content.tar.gz.
	dir := filepath.Dir(contentPath)
	assert.FileExists(t, filepath.Join(dir, "config.json"))

	// Verify the directory structure: {cacheDir}/complypacks/{evaluator-id}/{version}/
	expectedDir := filepath.Join(cacheDir, "complypacks", "io.complytime.opa", "1.0.0")
	assert.Equal(t, expectedDir, dir)
}

func TestComplypackCache_Store_AtomicWrite(t *testing.T) {
	cacheDir := t.TempDir()
	cc := cache.NewComplypackCache(cacheDir)

	// First store succeeds — establishes a valid cache entry.
	cfg := newTestConfig("io.complytime.opa", "1.0.0")
	_, err := cc.Store(cfg, strings.NewReader("good content"))
	require.NoError(t, err)

	// Second store with an invalid evaluator-id should fail validation
	// before writing anything. This verifies that a failed Store call
	// does not leave partial artifacts at the final cache path.
	badCfg := newTestConfig("../../evil", "1.0.0")
	_, err = cc.Store(badCfg, strings.NewReader("bad content"))
	require.Error(t, err)

	// The original entry must still be intact — no partial overwrite.
	expectedDir := filepath.Join(cacheDir, "complypacks", "io.complytime.opa", "1.0.0")
	assert.FileExists(t, filepath.Join(expectedDir, "content.tar.gz"))
	assert.FileExists(t, filepath.Join(expectedDir, "config.json"))

	// The evil path must not exist at all.
	evilDir := filepath.Join(cacheDir, "complypacks", "../../evil", "1.0.0")
	assert.NoDirExists(t, evilDir)
}

func TestComplypackCache_Store_MultipleVersions(t *testing.T) {
	cacheDir := t.TempDir()
	cc := cache.NewComplypackCache(cacheDir)

	cfg1 := newTestConfig("io.complytime.opa", "1.0.0")
	path1, err := cc.Store(cfg1, strings.NewReader("v1 content"))
	require.NoError(t, err)

	cfg2 := newTestConfig("io.complytime.opa", "2.0.0")
	path2, err := cc.Store(cfg2, strings.NewReader("v2 content"))
	require.NoError(t, err)

	// Both versions must exist independently.
	assert.FileExists(t, path1)
	assert.FileExists(t, path2)

	// Verify they are in separate directories.
	assert.DirExists(t, filepath.Join(cacheDir, "complypacks", "io.complytime.opa", "1.0.0"))
	assert.DirExists(t, filepath.Join(cacheDir, "complypacks", "io.complytime.opa", "2.0.0"))

	// Verify content is distinct.
	data1, err := os.ReadFile(path1)
	require.NoError(t, err)
	data2, err := os.ReadFile(path2)
	require.NoError(t, err)
	assert.Equal(t, "v1 content", string(data1))
	assert.Equal(t, "v2 content", string(data2))
}

func TestComplypackCache_Lookup_Found(t *testing.T) {
	cacheDir := t.TempDir()
	cc := cache.NewComplypackCache(cacheDir)

	cfg := newTestConfig("io.complytime.opa", "1.0.0")
	storedPath, err := cc.Store(cfg, strings.NewReader("test content"))
	require.NoError(t, err)

	contentPath, returnedCfg, err := cc.Lookup("io.complytime.opa", "1.0.0")
	require.NoError(t, err)

	// Verify the returned path matches what Store returned.
	assert.Equal(t, storedPath, contentPath)

	// Verify the returned config matches what was stored.
	assert.Equal(t, "io.complytime.opa", returnedCfg.EvaluatorID)
	assert.Equal(t, "1.0.0", returnedCfg.Version)
}

func TestComplypackCache_Lookup_NotFound(t *testing.T) {
	cacheDir := t.TempDir()
	cc := cache.NewComplypackCache(cacheDir)

	_, _, err := cc.Lookup("io.complytime.opa", "9.9.9")
	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist,
		"lookup for missing complypack should wrap os.ErrNotExist")
}

// --- LookupByEvaluatorID tests ---

func TestComplypackCache_LookupByEvaluatorID_NotFound(t *testing.T) {
	cacheDir := t.TempDir()
	cc := cache.NewComplypackCache(cacheDir)

	// Lookup an evaluator that was never stored — should return empty path,
	// nil config, nil error (non-error "not found" contract).
	contentPath, cfg, err := cc.LookupByEvaluatorID("io.complytime.nonexistent")
	require.NoError(t, err, "missing evaluator should not return an error")
	assert.Empty(t, contentPath, "content path should be empty for missing evaluator")
	assert.Nil(t, cfg, "config should be nil for missing evaluator")
}

func TestComplypackCache_LookupByEvaluatorID_SkipsHiddenDirs(t *testing.T) {
	cacheDir := t.TempDir()
	cc := cache.NewComplypackCache(cacheDir)

	// Create a hidden directory that simulates an in-progress atomic write.
	// This mimics the .complypack-tmp-xxx directories created by Store().
	evalDir := filepath.Join(cacheDir, "complypacks", "io.complytime.opa")
	hiddenDir := filepath.Join(evalDir, ".complypack-tmp-abc123")
	require.NoError(t, os.MkdirAll(hiddenDir, 0755))

	// Place a content.tar.gz inside the hidden dir so it would match if
	// the hidden-dir filter were missing.
	require.NoError(t, os.WriteFile(
		filepath.Join(hiddenDir, "content.tar.gz"),
		[]byte("partial content"),
		0600,
	))

	// LookupByEvaluatorID must skip the hidden directory and return empty
	// results since no real version directory exists.
	contentPath, cfg, err := cc.LookupByEvaluatorID("io.complytime.opa")
	require.NoError(t, err, "hidden dirs should be silently skipped")
	assert.Empty(t, contentPath, "content path should be empty when only hidden dirs exist")
	assert.Nil(t, cfg, "config should be nil when only hidden dirs exist")
}

func TestComplypackCache_LookupByEvaluatorID_InvalidInput(t *testing.T) {
	cacheDir := t.TempDir()
	cc := cache.NewComplypackCache(cacheDir)

	// Path traversal input must be rejected by ValidatePathComponent.
	_, _, err := cc.LookupByEvaluatorID("../../evil")
	require.Error(t, err, "path traversal evaluator-id must be rejected")
	assert.Contains(t, err.Error(), "invalid evaluator-id")
}

// --- State complypack round-trip tests ---

func TestState_ComplypackRoundTrip(t *testing.T) {
	stateDir := t.TempDir()

	// Create state with complypack entries.
	state, err := cache.LoadState(stateDir)
	require.NoError(t, err)

	state.UpdateComplypackState("io.complytime.opa", "1.0.0", "sha256:abc123")
	state.UpdateComplypackState("io.complytime.kyverno", "2.0.0", "sha256:def456")

	err = cache.SaveState(state, stateDir)
	require.NoError(t, err)

	// Reload and verify.
	loaded, err := cache.LoadState(stateDir)
	require.NoError(t, err)

	ps1, ok := loaded.GetComplypackState("io.complytime.opa")
	require.True(t, ok, "expected io.complytime.opa to be present in loaded state")
	assert.Equal(t, "1.0.0", ps1.Version)
	assert.Equal(t, "sha256:abc123", ps1.Digest)
	assert.WithinDuration(t, time.Now(), ps1.LastUpdated, 5*time.Second)

	ps2, ok := loaded.GetComplypackState("io.complytime.kyverno")
	require.True(t, ok, "expected io.complytime.kyverno to be present in loaded state")
	assert.Equal(t, "2.0.0", ps2.Version)
	assert.Equal(t, "sha256:def456", ps2.Digest)
	assert.WithinDuration(t, time.Now(), ps2.LastUpdated, 5*time.Second)
}

func TestState_ComplypackLoadMissing_ReturnsEmpty(t *testing.T) {
	stateDir := t.TempDir()

	// Load from a directory with no state.json — should return empty but not nil.
	state, err := cache.LoadState(stateDir)
	require.NoError(t, err)

	assert.NotNil(t, state.Complypacks, "Complypacks map must not be nil")
	assert.Empty(t, state.Complypacks, "Complypacks map must be empty")

	// Verify no complypack state is found.
	_, ok := state.GetComplypackState("io.complytime.opa")
	assert.False(t, ok, "expected no complypack state for missing file")
}

// TestState_ComplypackLoadLegacy_InitializesMap verifies that loading a
// state.json written before the Complypacks field existed still initializes
// the Complypacks map to non-nil. This covers the nil-guard in LoadState.
func TestState_ComplypackLoadLegacy_InitializesMap(t *testing.T) {
	stateDir := t.TempDir()

	// Write a legacy state.json without the "complypacks" key.
	legacyJSON := `{
  "last_sync": "2025-01-01T00:00:00Z",
  "policies": {
    "nist": {"version": "v1.0", "digest": "sha256:abc", "last_updated": "2025-01-01T00:00:00Z"}
  }
}`
	statePath := filepath.Join(stateDir, "state.json")
	require.NoError(t, os.WriteFile(statePath, []byte(legacyJSON), 0600))

	state, err := cache.LoadState(stateDir)
	require.NoError(t, err)

	// Policies should be loaded from the file.
	ps, ok := state.GetPolicyState("nist")
	require.True(t, ok)
	assert.Equal(t, "v1.0", ps.Version)

	// Complypacks must be initialized to non-nil even though the key was absent.
	assert.NotNil(t, state.Complypacks, "Complypacks map must be initialized for legacy state files")
	assert.Empty(t, state.Complypacks)
}
