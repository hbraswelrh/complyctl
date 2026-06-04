// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTarGzFromFS_ValidFS(t *testing.T) {
	data := buildTarGzFromFS(opaComplypackData, "testdata/opa-complypack")

	// Verify the output is valid gzip
	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	// Verify the output is a valid tar archive with expected files
	tr := tar.NewReader(gr)
	files := make(map[string]bool)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		files[hdr.Name] = true
		assert.Equal(t, int64(0o644), hdr.Mode)
		assert.Greater(t, hdr.Size, int64(0))
	}

	assert.Contains(t, files, "complytime-mapping.json")
	assert.Contains(t, files, "run_as_nonroot.rego")
	assert.Contains(t, files, "resource_limits.rego")
	assert.Len(t, files, 3, "expected exactly 3 files in tar archive")
}

func TestBuildTarGzFromFS_ContentReadable(t *testing.T) {
	data := buildTarGzFromFS(opaComplypackData, "testdata/opa-complypack")

	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		content := make([]byte, hdr.Size)
		_, err = io.ReadFull(tr, content)
		require.NoError(t, err, "should be able to read full content of %s", hdr.Name)
		assert.NotEmpty(t, content)
	}
}

func TestBuildTarGzFromFS_AmpelFS(t *testing.T) {
	data := buildTarGzFromFS(ampelComplypackData, "testdata/ampel-complypack")

	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	files := make(map[string]bool)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		files[hdr.Name] = true
		assert.Equal(t, int64(0o644), hdr.Mode)
		assert.Greater(t, hdr.Size, int64(0))

		// Verify each file is valid JSON with a non-empty "id" field.
		content := make([]byte, hdr.Size)
		_, readErr := io.ReadFull(tr, content)
		require.NoError(t, readErr)

		var policy struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(content, &policy),
			"file %s should be valid JSON", hdr.Name)
		assert.NotEmpty(t, policy.ID,
			"file %s should have a non-empty id field", hdr.Name)
	}

	assert.Contains(t, files, "block-force-push.json")
	assert.Len(t, files, 1, "expected exactly 1 file in tar archive")
}

func TestBuildDummyTarGz_SingleFile(t *testing.T) {
	content := []byte(`{"test": true}`)
	data := buildDummyTarGz("test.json", content)

	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	hdr, err := tr.Next()
	require.NoError(t, err)

	assert.Equal(t, "test.json", hdr.Name)
	assert.Equal(t, int64(len(content)), hdr.Size)

	read := make([]byte, hdr.Size)
	_, err = io.ReadFull(tr, read)
	require.NoError(t, err)
	assert.Equal(t, content, read)

	// Should be only one file
	_, err = tr.Next()
	assert.Equal(t, io.EOF, err)
}

func TestSeedDefaults_AllReposSeeded(t *testing.T) {
	store := newContentStore()
	store.seedDefaults()

	// Verify all expected repositories are seeded
	repos := store.listRepositories()
	assert.Contains(t, repos, "complypacks/ampel-bp")
	assert.Contains(t, repos, "complypacks/test-opa-complypack")
	assert.Contains(t, repos, "policies/ampel-branch-protection")
	assert.Contains(t, repos, "policies/test-branch-protection")
	assert.Contains(t, repos, "policies/test-opa-policy")

	// Verify OPA policy has expected tags
	opaRepo := store.repos["policies/test-opa-policy"]
	require.NotNil(t, opaRepo)
	_, hasLatest := opaRepo.tags["latest"]
	assert.True(t, hasLatest, "OPA policy should have 'latest' tag")
	_, hasV1 := opaRepo.tags["v1.0.0"]
	assert.True(t, hasV1, "OPA policy should have 'v1.0.0' tag")

	// Verify OPA complypack has expected tags
	complypackRepo := store.repos["complypacks/test-opa-complypack"]
	require.NotNil(t, complypackRepo)
	_, hasLatest = complypackRepo.tags["latest"]
	assert.True(t, hasLatest, "OPA complypack should have 'latest' tag")

	// Verify ampel complypack has expected tags and valid content
	ampelRepo := store.repos["complypacks/ampel-bp"]
	require.NotNil(t, ampelRepo)
	_, hasLatest = ampelRepo.tags["latest"]
	assert.True(t, hasLatest, "ampel complypack should have 'latest' tag")
	_, hasV1 = ampelRepo.tags["v1.0.0"]
	assert.True(t, hasV1, "ampel complypack should have 'v1.0.0' tag")

	// Verify the ampel complypack content blob contains valid granular policy JSON.
	ampelArt := ampelRepo.tags["latest"]
	require.NotNil(t, ampelArt)
	var ampelManifest ociManifest
	require.NoError(t, json.Unmarshal(ampelArt.manifestBytes, &ampelManifest))
	require.NotEmpty(t, ampelManifest.Layers, "ampel complypack should have at least one layer")

	contentDigest := ampelManifest.Layers[0].Digest
	contentBlob, ok := ampelRepo.blobs[contentDigest]
	require.True(t, ok, "content blob should exist for digest %s", contentDigest)

	gr, err := gzip.NewReader(bytes.NewReader(contentBlob.data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	fileCount := 0
	for {
		hdr, tarErr := tr.Next()
		if tarErr == io.EOF {
			break
		}
		require.NoError(t, tarErr)
		fileCount++

		data := make([]byte, hdr.Size)
		_, readErr := io.ReadFull(tr, data)
		require.NoError(t, readErr)

		var policy struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(data, &policy),
			"ampel complypack file %s should be valid JSON", hdr.Name)
		assert.NotEmpty(t, policy.ID,
			"ampel complypack file %s should have a non-empty id field", hdr.Name)
	}
	assert.GreaterOrEqual(t, fileCount, 1,
		"ampel complypack should contain at least one file")
}

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
