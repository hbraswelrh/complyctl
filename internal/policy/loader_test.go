// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/complytime/complyctl/internal/cache"
	"github.com/complytime/complyctl/internal/cache/cachetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- T155: PolicyExists, ResolveVersion, GetCachedVersions ---

func TestPolicyExists_NonexistentPolicy(t *testing.T) {
	cacheDir := t.TempDir()
	cacheMgr := cache.NewCache(cacheDir)
	loader := NewLoader(cacheMgr)

	assert.False(t, loader.PolicyExists("nonexistent-policy", "v1"))
}

func TestPolicyExists_EmptyInputs(t *testing.T) {
	cacheDir := t.TempDir()
	cacheMgr := cache.NewCache(cacheDir)
	loader := NewLoader(cacheMgr)

	assert.False(t, loader.PolicyExists("", "v1"))
	assert.False(t, loader.PolicyExists("test", ""))
	assert.False(t, loader.PolicyExists("", ""))
}

func TestResolveVersion_EmptyCache(t *testing.T) {
	cacheDir := t.TempDir()
	cacheMgr := cache.NewCache(cacheDir)
	loader := NewLoader(cacheMgr)

	_, err := loader.ResolveVersion("nonexistent", "v1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in cache")
}

func TestResolveVersion_LatestFallbackEmptyCache(t *testing.T) {
	cacheDir := t.TempDir()
	cacheMgr := cache.NewCache(cacheDir)
	loader := NewLoader(cacheMgr)

	_, err := loader.ResolveVersion("nonexistent", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in cache")
}

func TestGetCachedVersions_NonexistentPolicy(t *testing.T) {
	cacheDir := t.TempDir()
	cacheMgr := cache.NewCache(cacheDir)
	loader := NewLoader(cacheMgr)

	versions, err := loader.GetCachedVersions("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, versions)
}

// --- T156: LoadLayerByMediaType ---

func TestLoadLayerByMediaType_EmptyPolicyID(t *testing.T) {
	cacheDir := t.TempDir()
	cacheMgr := cache.NewCache(cacheDir)
	loader := NewLoader(cacheMgr)

	_, err := loader.LoadLayerByMediaType("", "v1", "application/vnd.gemara.policy.v1+yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "policy ID cannot be empty")
}

func TestLoadLayerByMediaType_EmptyVersion(t *testing.T) {
	cacheDir := t.TempDir()
	cacheMgr := cache.NewCache(cacheDir)
	loader := NewLoader(cacheMgr)

	_, err := loader.LoadLayerByMediaType("test", "", "application/vnd.gemara.policy.v1+yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version cannot be empty")
}

func TestLoadLayerByMediaType_EmptyMediaType(t *testing.T) {
	cacheDir := t.TempDir()
	cacheMgr := cache.NewCache(cacheDir)
	loader := NewLoader(cacheMgr)

	_, err := loader.LoadLayerByMediaType("test", "v1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "media type cannot be empty")
}

func TestLoadLayerByMediaType_NonexistentPolicy(t *testing.T) {
	cacheDir := t.TempDir()
	cacheMgr := cache.NewCache(cacheDir)
	loader := NewLoader(cacheMgr)

	_, err := loader.LoadLayerByMediaType("missing", "v1", "application/vnd.gemara.policy.v1+yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in cache")
}

// seedTestPolicy syncs a mock policy into the cache and returns the loader.
func seedTestPolicy(t *testing.T, policyID, version string) *Loader {
	t.Helper()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	mock := cachetest.NewMockPolicySource()
	mock.SeedPolicy(policyID, version, "sha256:test-digest")

	cacheMgr := cache.NewCache(cacheDir)
	state, err := cache.LoadState(cacheDir)
	require.NoError(t, err)

	sync := cache.NewSync(cacheMgr, state, mock)
	require.NoError(t, sync.SyncPolicy(context.Background(), policyID, "latest"))

	return NewLoader(cacheMgr)
}

func TestLoadLayerByMediaType_HappyPath(t *testing.T) {
	loader := seedTestPolicy(t, "test-policy", "v1.0.0")

	data, err := loader.LoadLayerByMediaType("test-policy", "v1.0.0", "application/vnd.gemara.policy.v1+yaml")
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "test-policy")
}

func TestLoadLayerByMediaType_WrongMediaType(t *testing.T) {
	loader := seedTestPolicy(t, "test-policy", "v1.0.0")

	_, err := loader.LoadLayerByMediaType("test-policy", "v1.0.0", "application/vnd.gemara.catalog.v1+yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoadLayerByMediaType_WrongVersion(t *testing.T) {
	loader := seedTestPolicy(t, "test-policy", "v1.0.0")

	_, err := loader.LoadLayerByMediaType("test-policy", "v2.0.0", "application/vnd.gemara.policy.v1+yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in cache")
}
