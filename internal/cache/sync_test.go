// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/complytime/complyctl/internal/cache"
	"github.com/complytime/complyctl/internal/cache/cachetest"
)

func TestSync_CopyOnSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	mock := cachetest.NewMockPolicySource()
	mock.SeedPolicy("test-policy", "v1.0.0", "sha256:abc123")

	cacheMgr := cache.NewCache(cacheDir)
	state, err := cache.LoadState(cacheDir)
	require.NoError(t, err)

	sync := cache.NewSync(cacheMgr, state, mock)

	err = sync.SyncPolicy(context.Background(), "test-policy", "latest")
	require.NoError(t, err)

	// Verify state was updated
	state2, err := cache.LoadState(cacheDir)
	require.NoError(t, err)
	ps, ok := state2.GetPolicyState("test-policy")
	assert.True(t, ok)
	assert.NotEmpty(t, ps.Digest)
	assert.NotEmpty(t, ps.Version)

	// Verify OCI Layout store exists (oci-layout marker and index.json)
	storePath := cacheMgr.PolicyStorePath("test-policy")
	assert.FileExists(t, filepath.Join(storePath, "oci-layout"))
	assert.FileExists(t, filepath.Join(storePath, "index.json"))
	assert.DirExists(t, filepath.Join(storePath, "blobs", "sha256"))
}

func TestSync_CopyOnSuccess_PinnedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	mock := cachetest.NewMockPolicySource()
	mock.SeedPolicy("test-policy", "v1.0.0", "sha256:abc123")

	cacheMgr := cache.NewCache(cacheDir)
	state, err := cache.LoadState(cacheDir)
	require.NoError(t, err)

	sync := cache.NewSync(cacheMgr, state, mock)

	err = sync.SyncPolicy(context.Background(), "test-policy", "v1.0.0")
	require.NoError(t, err)

	assert.Equal(t, "test-policy:v1.0.0", mock.LastLookupRef,
		"source should receive the versioned ref when a pinned version is provided")

	state2, err := cache.LoadState(cacheDir)
	require.NoError(t, err)
	ps, ok := state2.GetPolicyState("test-policy")
	assert.True(t, ok)
	assert.Equal(t, "v1.0.0", ps.Version)
	assert.Equal(t, "sha256:abc123", ps.Digest)
}

func TestSync_FailureOnMissingPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	mock := cachetest.NewMockPolicySource()

	cacheMgr := cache.NewCache(cacheDir)
	state, err := cache.LoadState(cacheDir)
	require.NoError(t, err)

	sync := cache.NewSync(cacheMgr, state, mock)

	err = sync.SyncPolicy(context.Background(), "nonexistent-policy", "latest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSync_IncrementalSkip(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	mock := cachetest.NewMockPolicySource()
	mock.SeedPolicy("test-policy", "v1.0.0", "sha256:abc123")

	cacheMgr := cache.NewCache(cacheDir)
	state, err := cache.LoadState(cacheDir)
	require.NoError(t, err)

	sync := cache.NewSync(cacheMgr, state, mock)

	// First sync
	err = sync.SyncPolicy(context.Background(), "test-policy", "latest")
	require.NoError(t, err)

	state2, err := cache.LoadState(cacheDir)
	require.NoError(t, err)
	ps, ok := state2.GetPolicyState("test-policy")
	require.True(t, ok)
	originalDigest := ps.Digest

	// Second sync with same digest — should be no-op (FR-004)
	sync2 := cache.NewSync(cacheMgr, state2, mock)
	err = sync2.SyncPolicy(context.Background(), "test-policy", "latest")
	require.NoError(t, err)

	state3, err := cache.LoadState(cacheDir)
	require.NoError(t, err)
	ps3, _ := state3.GetPolicyState("test-policy")
	assert.Equal(t, originalDigest, ps3.Digest, "digest should not change for incremental sync")
}

func TestSync_EmptyPolicyID(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	mock := cachetest.NewMockPolicySource()
	cacheMgr := cache.NewCache(cacheDir)
	state, _ := cache.LoadState(cacheDir)

	sync := cache.NewSync(cacheMgr, state, mock)
	err := sync.SyncPolicy(context.Background(), "", "latest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "policy ID cannot be empty")
}

func TestSync_RedownloadAfterDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	mock := cachetest.NewMockPolicySource()
	mock.SeedPolicy("test-policy", "v1.0.0", "sha256:abc123")

	cacheMgr := cache.NewCache(cacheDir)
	state, err := cache.LoadState(cacheDir)
	require.NoError(t, err)

	sync := cache.NewSync(cacheMgr, state, mock)

	err = sync.SyncPolicy(context.Background(), "test-policy", "latest")
	require.NoError(t, err)

	storePath := cacheMgr.PolicyStorePath("test-policy")
	assert.FileExists(t, filepath.Join(storePath, "oci-layout"))

	require.NoError(t, os.RemoveAll(storePath))
	assert.NoDirExists(t, storePath)

	state2, err := cache.LoadState(cacheDir)
	require.NoError(t, err)
	sync2 := cache.NewSync(cacheMgr, state2, mock)

	err = sync2.SyncPolicy(context.Background(), "test-policy", "latest")
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(storePath, "oci-layout"))
	assert.DirExists(t, filepath.Join(storePath, "blobs", "sha256"))
}

// TestSync_StressConcurrentFailures performs 100 sync iterations with alternating
// success and failure scenarios, verifying zero cache corruption (SC-008/FR-006).
func TestSync_StressConcurrentFailures(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	const iterations = 100
	policyID := "stress-test-policy"

	mock := cachetest.NewMockPolicySource()
	mock.SeedPolicy(policyID, "v1.0.0", "sha256:stress123")

	cacheMgr := cache.NewCache(cacheDir)

	successCount := 0
	failCount := 0

	for i := 0; i < iterations; i++ {
		state, loadErr := cache.LoadState(cacheDir)
		require.NoError(t, loadErr, "state load must not fail on iteration %d", i)

		syncMgr := cache.NewSync(cacheMgr, state, mock)

		if i%3 == 0 {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			syncErr := syncMgr.SyncPolicy(ctx, policyID, "latest")
			if syncErr != nil {
				failCount++
			} else {
				successCount++
			}
		} else if i%7 == 0 {
			syncErr := syncMgr.SyncPolicy(context.Background(),
				fmt.Sprintf("nonexistent-%d", i), "latest")
			require.Error(t, syncErr, "nonexistent policy must fail on iteration %d", i)
			failCount++
		} else {
			syncErr := syncMgr.SyncPolicy(context.Background(), policyID, "latest")
			require.NoError(t, syncErr, "normal sync must not fail on iteration %d", i)
			successCount++
		}
	}

	t.Logf("Stress test complete: %d success, %d failure out of %d iterations",
		successCount, failCount, iterations)

	// Final: cache state must be loadable and consistent
	finalState, err := cache.LoadState(cacheDir)
	require.NoError(t, err, "final state must load without error")

	ps, ok := finalState.GetPolicyState(policyID)
	if ok {
		assert.NotEmpty(t, ps.Digest, "successful policy must have a digest")
		assert.NotEmpty(t, ps.Version, "successful policy must have a version")

		// Verify OCI Layout store integrity
		storePath := cacheMgr.PolicyStorePath(policyID)
		assert.FileExists(t, filepath.Join(storePath, "oci-layout"),
			"OCI layout marker must exist after successful syncs")
	}
}

func TestBuildLookupRef(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		tag        string
		digest     string
		want       string
	}{
		{"tag version", "org/policy", "v1.0", "", "org/policy:v1.0"},
		{"sha256 digest", "org/policy", "", "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", "org/policy@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"},
		{"sha512 digest", "org/policy", "", "sha512:def456", "org/policy@sha512:def456"},
		{"empty version", "org/policy", "", "", "org/policy"},
		{"latest version", "org/policy", "latest", "", "org/policy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cache.BuildLookupRef(tt.repository, tt.tag, tt.digest)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBuildLookupRef_DigestPrecedence verifies that when both tag and
// digest are provided, digest takes precedence (OCI convention).
func TestBuildLookupRef_DigestPrecedence(t *testing.T) {
	got := cache.BuildLookupRef("org/policy", "v1.0", "sha256:abc123")
	assert.Equal(t, "org/policy@sha256:abc123", got)
}

// TestBuildLookupRef_SHA384Digest verifies sha384 digests are handled
// correctly through the typed fields.
func TestBuildLookupRef_SHA384Digest(t *testing.T) {
	sha384Digest := "sha384:" + "a" + strings.Repeat("b", 95)
	got := cache.BuildLookupRef("org/policy", "", sha384Digest)
	assert.Equal(t, "org/policy@"+sha384Digest, got)
}

// TestSync_SHA384Digest verifies that a sha384 digest version string is
// correctly classified and used as a digest (not a tag) in the sync path.
func TestSync_SHA384Digest(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	sha384Digest := "sha384:" + "a" + strings.Repeat("b", 95)
	mock := cachetest.NewMockPolicySource()
	mock.SeedPolicy("test-policy", "v1.0.0", "sha256:abc123")

	cacheMgr := cache.NewCache(cacheDir)
	state, err := cache.LoadState(cacheDir)
	require.NoError(t, err)

	sync := cache.NewSync(cacheMgr, state, mock)

	// Pass a sha384 digest as the version string. classifyVersion must
	// detect it as a digest and BuildLookupRef must use "@" separator.
	_ = sync.SyncPolicy(context.Background(), "test-policy", sha384Digest)

	assert.Contains(t, mock.LastLookupRef, "@"+sha384Digest,
		"sha384 digest must use @ separator, not : separator")
	assert.NotContains(t, mock.LastLookupRef, ":"+sha384Digest,
		"sha384 digest must not be treated as a tag")
}

// TestBuildLookupRef_Regression_NoDoubleTag verifies that the original
// bug (issue #594) is prevented: a :tag in the repository must not
// produce a double-tagged reference like "repo:v0.4.0:v0.4.0".
func TestBuildLookupRef_Regression_NoDoubleTag(t *testing.T) {
	// When ParsePolicyRef correctly extracts the tag, Repository
	// will be "complytime/complypack-ampel-bp" and Tag "v0.4.0".
	// BuildLookupRef should produce a single-tagged reference.
	lookupRef := cache.BuildLookupRef("complytime/complypack-ampel-bp", "v0.4.0", "")
	assert.Equal(t, "complytime/complypack-ampel-bp:v0.4.0", lookupRef)
	assert.NotContains(t, lookupRef, ":v0.4.0:v0.4.0")
}
