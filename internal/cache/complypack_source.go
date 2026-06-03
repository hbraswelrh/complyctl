// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"encoding/json"
	"fmt"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	ocistore "oras.land/oras-go/v2/content/oci"

	"github.com/complytime/complyctl/internal/registry"
	"github.com/complytime/complypack/pkg/complypack"
)

// ComplypackSource abstracts remote complypack access for sync operations.
// Mirrors PolicySource but adds artifact type verification to ensure the
// fetched OCI artifact is a valid complypack.
type ComplypackSource interface {
	// DefinitionVersion resolves the remote manifest digest and version tag
	// for a complypack identified by repository path.
	DefinitionVersion(ctx context.Context, repository string) (digest string, version string, err error)

	// CopyComplypack copies the remote complypack artifact into a local OCI
	// Layout store, verifies the manifest artifactType matches
	// complypack.MediaTypeArtifact, then unpacks the config and content.
	// Caller must close the returned ComplyPack.Content.
	CopyComplypack(ctx context.Context, repository, tag string, dst *ocistore.Store) (ocispec.Descriptor, error)
}

// RegistryComplypackSource wraps a registry.Client to implement ComplypackSource.
// Uses oras.Copy() for atomic remote-to-local transfer with digest verification,
// then validates the OCI artifactType before returning.
type RegistryComplypackSource struct {
	client *registry.Client
}

// NewRegistryComplypackSource creates a ComplypackSource backed by a live OCI registry.
func NewRegistryComplypackSource(client *registry.Client) *RegistryComplypackSource {
	return &RegistryComplypackSource{client: client}
}

func (s *RegistryComplypackSource) DefinitionVersion(ctx context.Context, repository string) (string, string, error) {
	return s.client.DefinitionVersion(ctx, repository)
}

// CopyComplypack copies the remote artifact into dst and verifies the manifest
// artifactType matches complypack.MediaTypeArtifact. Returns the manifest
// descriptor on success.
func (s *RegistryComplypackSource) CopyComplypack(ctx context.Context, repository, tag string, dst *ocistore.Store) (ocispec.Descriptor, error) {
	repo, err := s.client.NewRemoteRepository(ctx, repository)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to create remote repository: %w", err)
	}

	desc, err := oras.Copy(ctx, repo, tag, dst, tag, oras.CopyOptions{})
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to copy complypack artifact: %w", err)
	}

	return VerifyComplypackArtifactType(ctx, desc, dst)
}

// VerifyComplypackArtifactType checks that the descriptor's artifactType matches
// complypack.MediaTypeArtifact. If desc.ArtifactType is empty (registry did not
// populate it via OCI Distribution Spec v1.1 headers), the function falls back to
// reading and parsing the manifest body from the local store. On success, the
// returned descriptor has ArtifactType set.
func VerifyComplypackArtifactType(ctx context.Context, desc ocispec.Descriptor, store content.ReadOnlyStorage) (ocispec.Descriptor, error) {
	artifactType := desc.ArtifactType
	if artifactType == "" {
		manifestData, err := store.Fetch(ctx, desc)
		if err != nil {
			return ocispec.Descriptor{}, fmt.Errorf("failed to read manifest from local store: %w", err)
		}
		defer manifestData.Close()

		var manifest struct {
			ArtifactType string `json:"artifactType"`
		}
		if err := json.NewDecoder(manifestData).Decode(&manifest); err != nil {
			return ocispec.Descriptor{}, fmt.Errorf("failed to parse manifest: %w", err)
		}
		artifactType = manifest.ArtifactType
	}

	if artifactType != complypack.MediaTypeArtifact {
		return ocispec.Descriptor{}, fmt.Errorf(
			"artifact type mismatch: got %q, want %q",
			artifactType, complypack.MediaTypeArtifact,
		)
	}

	desc.ArtifactType = artifactType
	return desc, nil
}
