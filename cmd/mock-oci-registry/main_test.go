// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveContentDir_Default(t *testing.T) {
	t.Setenv("MOCK_REGISTRY_CONTENT_DIR", "")
	assert.Equal(t, defaultContentDir, resolveContentDir())
}

func TestResolveContentDir_EnvOverride(t *testing.T) {
	t.Setenv("MOCK_REGISTRY_CONTENT_DIR", "/custom/path")
	assert.Equal(t, "/custom/path", resolveContentDir())
}

func TestSeedFromDirectory_ValidPolicies(t *testing.T) {
	dir := t.TempDir()

	// Create a valid policy directory with catalog.yaml and policy.yaml
	policyDir := filepath.Join(dir, "my-policy")
	require.NoError(t, os.Mkdir(policyDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(policyDir, "catalog.yaml"),
		[]byte("catalog: test-content"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(policyDir, "policy.yaml"),
		[]byte("policy: test-content"),
		0o600,
	))

	store := newContentStore()
	store.seedFromDirectory(dir)

	repo, ok := store.repos["policies/my-policy"]
	require.True(t, ok, "repository policies/my-policy should exist")

	art, ok := repo.tags["latest"]
	require.True(t, ok, "tag 'latest' should exist")
	assert.NotEmpty(t, art.manifestDigest)
	assert.NotEmpty(t, art.manifestBytes)

	// Verify two layer blobs + one config blob = 3 blobs
	assert.Len(t, repo.blobs, 3)
}

func TestSeedFromDirectory_MultiplePolicies(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"policy-a", "policy-b"} {
		policyDir := filepath.Join(dir, name)
		require.NoError(t, os.Mkdir(policyDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(policyDir, "catalog.yaml"),
			[]byte("catalog: "+name),
			0o600,
		))
		require.NoError(t, os.WriteFile(
			filepath.Join(policyDir, "policy.yaml"),
			[]byte("policy: "+name),
			0o600,
		))
	}

	store := newContentStore()
	store.seedFromDirectory(dir)

	assert.Contains(t, store.repos, "policies/policy-a")
	assert.Contains(t, store.repos, "policies/policy-b")
}

func TestSeedFromDirectory_SkipsMissingCatalog(t *testing.T) {
	dir := t.TempDir()

	// Directory with only policy.yaml — no catalog.yaml
	policyDir := filepath.Join(dir, "incomplete")
	require.NoError(t, os.Mkdir(policyDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(policyDir, "policy.yaml"),
		[]byte("policy: data"),
		0o600,
	))

	store := newContentStore()
	store.seedFromDirectory(dir)

	assert.NotContains(t, store.repos, "policies/incomplete")
}

func TestSeedFromDirectory_SkipsMissingPolicy(t *testing.T) {
	dir := t.TempDir()

	// Directory with only catalog.yaml — no policy.yaml
	policyDir := filepath.Join(dir, "catalog-only")
	require.NoError(t, os.Mkdir(policyDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(policyDir, "catalog.yaml"),
		[]byte("catalog: data"),
		0o600,
	))

	store := newContentStore()
	store.seedFromDirectory(dir)

	assert.NotContains(t, store.repos, "policies/catalog-only")
}

func TestSeedFromDirectory_SkipsFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a regular file (not a directory) — should be skipped
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "README.md"),
		[]byte("not a policy"),
		0o600,
	))

	store := newContentStore()
	store.seedFromDirectory(dir)

	assert.Empty(t, store.repos)
}

func TestSeedFromDirectory_NonexistentDirectory(t *testing.T) {
	store := newContentStore()

	// Should not panic or error — just a no-op
	store.seedFromDirectory("/nonexistent/path/that/does/not/exist")

	assert.Empty(t, store.repos)
}

func TestSeedFromDirectory_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	store := newContentStore()
	store.seedFromDirectory(dir)

	assert.Empty(t, store.repos)
}

func TestSeedFromDirectory_SkipsInvalidNames(t *testing.T) {
	dir := t.TempDir()

	// Directory with dots in the name — should be rejected
	policyDir := filepath.Join(dir, "bad.name")
	require.NoError(t, os.Mkdir(policyDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(policyDir, "catalog.yaml"),
		[]byte("catalog: data"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(policyDir, "policy.yaml"),
		[]byte("policy: data"),
		0o600,
	))

	store := newContentStore()
	store.seedFromDirectory(dir)

	assert.NotContains(t, store.repos, "policies/bad.name")
}

func TestSeedFromDirectory_SkipsSymlinks(t *testing.T) {
	dir := t.TempDir()

	// Create a real policy directory
	realDir := filepath.Join(dir, "real-policy")
	require.NoError(t, os.Mkdir(realDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(realDir, "catalog.yaml"),
		[]byte("catalog: data"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(realDir, "policy.yaml"),
		[]byte("policy: data"),
		0o600,
	))

	// Create a symlink pointing to the real directory
	symlinkDir := filepath.Join(dir, "symlinked-policy")
	require.NoError(t, os.Symlink(realDir, symlinkDir))

	store := newContentStore()
	store.seedFromDirectory(dir)

	// Real directory should be seeded, symlink should be skipped
	assert.Contains(t, store.repos, "policies/real-policy")
	assert.NotContains(t, store.repos, "policies/symlinked-policy")
}

func TestSeedFromDirectory_RejectsOversizedFiles(t *testing.T) {
	dir := t.TempDir()

	policyDir := filepath.Join(dir, "big-policy")
	require.NoError(t, os.Mkdir(policyDir, 0o755))

	// Create a catalog that exceeds maxPolicyFileSize (10 MB)
	bigData := make([]byte, maxPolicyFileSize+1)
	require.NoError(t, os.WriteFile(
		filepath.Join(policyDir, "catalog.yaml"),
		bigData,
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(policyDir, "policy.yaml"),
		[]byte("policy: data"),
		0o600,
	))

	store := newContentStore()
	store.seedFromDirectory(dir)

	assert.NotContains(t, store.repos, "policies/big-policy")
}

func TestReadFileLimited_RejectsSymlink(t *testing.T) {
	dir := t.TempDir()

	// Create a real file and a symlink to it
	realFile := filepath.Join(dir, "real.yaml")
	require.NoError(t, os.WriteFile(realFile, []byte("data"), 0o600))

	linkFile := filepath.Join(dir, "link.yaml")
	require.NoError(t, os.Symlink(realFile, linkFile))

	_, err := readFileLimited(linkFile, maxPolicyFileSize)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
}

func TestSeedFromDirectory_OverwritesExistingRepo(t *testing.T) {
	dir := t.TempDir()

	// Create a policy with the same name as the default
	policyDir := filepath.Join(dir, "test-branch-protection")
	require.NoError(t, os.Mkdir(policyDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(policyDir, "catalog.yaml"),
		[]byte("catalog: override"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(policyDir, "policy.yaml"),
		[]byte("policy: override"),
		0o600,
	))

	store := newContentStore()
	store.seedDefaults()

	// Capture the default artifact digest before seeding from directory
	defaultArt := store.repos["policies/test-branch-protection"].tags["latest"]
	defaultDigest := defaultArt.manifestDigest

	store.seedFromDirectory(dir)

	// The directory seed overwrites — verify the digest changed
	newArt := store.repos["policies/test-branch-protection"].tags["latest"]
	assert.NotEqual(t, defaultDigest, newArt.manifestDigest,
		"directory seed should overwrite existing repo")
}
