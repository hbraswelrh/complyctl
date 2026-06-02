// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ocistore "oras.land/oras-go/v2/content/oci"

	"github.com/complytime/complyctl/internal/cache"
)

// seedTestLayout creates a minimal OCI Layout at the given path with a single
// layer and a tagged manifest. Returns the manifest digest.
func seedTestLayout(t *testing.T, layoutPath, tag string) string {
	t.Helper()
	ctx := context.Background()

	store, err := ocistore.New(layoutPath)
	require.NoError(t, err)

	// Push empty config.
	configData := []byte("{}")
	configDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeEmptyJSON,
		Digest:    digest.FromBytes(configData),
		Size:      int64(len(configData)),
	}
	require.NoError(t, store.Push(ctx, configDesc, bytes.NewReader(configData)))

	// Push a test layer.
	layerData := []byte(`{"policy_id": "test-policy", "type": "test-layer"}`)
	layerDesc := ocispec.Descriptor{
		MediaType: "application/vnd.gemara.policy.v1+yaml",
		Digest:    digest.FromBytes(layerData),
		Size:      int64(len(layerData)),
	}
	require.NoError(t, store.Push(ctx, layerDesc, bytes.NewReader(layerData)))

	// Build and push manifest.
	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    configDesc,
		Layers:    []ocispec.Descriptor{layerDesc},
	}
	manifest.SchemaVersion = 2
	manifestData, err := json.Marshal(manifest)
	require.NoError(t, err)

	manifestDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestData),
		Size:      int64(len(manifestData)),
	}
	require.NoError(t, store.Push(ctx, manifestDesc, bytes.NewReader(manifestData)))
	require.NoError(t, store.Tag(ctx, manifestDesc, tag))

	return manifestDesc.Digest.String()
}

func TestLocalSource_DefinitionVersion_ResolvesDigest(t *testing.T) {
	srcDir := t.TempDir()
	expectedDigest := seedTestLayout(t, srcDir, "latest")

	src := cache.NewLocalSource(srcDir)
	gotDigest, gotVersion, err := src.DefinitionVersion(context.Background(), "test-policy")
	require.NoError(t, err)
	assert.Equal(t, expectedDigest, gotDigest)
	assert.Equal(t, "latest", gotVersion)
}

func TestLocalSource_DefinitionVersion_ExplicitTag(t *testing.T) {
	srcDir := t.TempDir()
	expectedDigest := seedTestLayout(t, srcDir, "v1.0.0")

	src := cache.NewLocalSource(srcDir)
	gotDigest, gotVersion, err := src.DefinitionVersion(context.Background(), "test-policy:v1.0.0")
	require.NoError(t, err)
	assert.Equal(t, expectedDigest, gotDigest)
	assert.Equal(t, "v1.0.0", gotVersion)
}

func TestLocalSource_CopyPolicy(t *testing.T) {
	ctx := context.Background()
	srcDir := t.TempDir()
	seedTestLayout(t, srcDir, "latest")

	dstDir := filepath.Join(t.TempDir(), "dst")
	dstStore, err := ocistore.New(dstDir)
	require.NoError(t, err)

	src := cache.NewLocalSource(srcDir)
	desc, err := src.CopyPolicy(ctx, "test-policy", "latest", dstStore)
	require.NoError(t, err)
	assert.Equal(t, ocispec.MediaTypeImageManifest, desc.MediaType)

	// Verify destination can resolve the tag.
	resolved, err := dstStore.Resolve(ctx, "latest")
	require.NoError(t, err)
	assert.Equal(t, desc.Digest, resolved.Digest)
}

func TestLocalSource_DefinitionVersion_NonexistentPath(t *testing.T) {
	src := cache.NewLocalSource("/nonexistent/path/to/layout")
	_, _, err := src.DefinitionVersion(context.Background(), "test-policy")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open local OCI layout")
}

func TestLocalSource_CopyPolicy_NonexistentPath(t *testing.T) {
	dstDir := filepath.Join(t.TempDir(), "dst")
	dstStore, err := ocistore.New(dstDir)
	require.NoError(t, err)

	src := cache.NewLocalSource("/nonexistent/path/to/layout")
	_, err = src.CopyPolicy(context.Background(), "test", "latest", dstStore)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open local OCI layout")
}

func TestLocalSource_DefinitionVersion_InvalidLayout(t *testing.T) {
	// Create a directory that exists but has no tagged manifests.
	// oci.New() on an empty directory creates a new OCI Layout, so the
	// error comes from store.Resolve() failing to find the tag.
	emptyDir := t.TempDir()

	src := cache.NewLocalSource(emptyDir)
	_, _, err := src.DefinitionVersion(context.Background(), "test-policy")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve tag")
}

func TestLocalSource_CopyPolicy_NonexistentTag(t *testing.T) {
	ctx := context.Background()
	srcDir := t.TempDir()
	seedTestLayout(t, srcDir, "latest")

	dstDir := filepath.Join(t.TempDir(), "dst")
	dstStore, err := ocistore.New(dstDir)
	require.NoError(t, err)

	src := cache.NewLocalSource(srcDir)
	_, err = src.CopyPolicy(ctx, "test-policy", "v99.0.0", dstStore)
	require.Error(t, err)
}
