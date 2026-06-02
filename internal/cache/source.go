// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"fmt"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	ocistore "oras.land/oras-go/v2/content/oci"

	"github.com/complytime/complyctl/internal/registry"
)

// PolicySource abstracts policy access for sync operations.
type PolicySource interface {
	DefinitionVersion(ctx context.Context, policyID string) (digest string, version string, err error)
	CopyPolicy(ctx context.Context, policyID, tag string, dst *ocistore.Store) (ocispec.Descriptor, error)
}

// RegistrySource wraps a registry.Client to implement PolicySource.
// Uses oras.Copy() for atomic remote-to-local transfer with digest verification.
type RegistrySource struct {
	client *registry.Client
}

func NewRegistrySource(client *registry.Client) *RegistrySource {
	return &RegistrySource{client: client}
}

func (s *RegistrySource) DefinitionVersion(ctx context.Context, policyID string) (string, string, error) {
	return s.client.DefinitionVersion(ctx, policyID)
}

func (s *RegistrySource) CopyPolicy(ctx context.Context, policyID, tag string, dst *ocistore.Store) (ocispec.Descriptor, error) {
	repo, err := s.client.NewRemoteRepository(ctx, policyID)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to create remote repository: %w", err)
	}
	return oras.Copy(ctx, repo, tag, dst, tag, oras.CopyOptions{})
}

// LocalSource implements PolicySource for local OCI Layout directories.
// Uses oras.Copy() for local-to-local transfer with digest verification.
type LocalSource struct {
	layoutPath string
}

// NewLocalSource creates a LocalSource that reads from the given OCI Layout
// directory path.
func NewLocalSource(layoutPath string) *LocalSource {
	return &LocalSource{layoutPath: layoutPath}
}

// DefinitionVersion reads the manifest digest and version tag from a local
// OCI Layout. The policyID may include a ":tag" suffix; if absent, "latest"
// is used.
func (s *LocalSource) DefinitionVersion(ctx context.Context, policyID string) (string, string, error) {
	store, err := ocistore.New(s.layoutPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open local OCI layout %s: %w", s.layoutPath, err)
	}

	tag := "latest"
	if parts := strings.SplitN(policyID, ":", 2); len(parts) == 2 {
		tag = parts[1]
	}

	desc, err := store.Resolve(ctx, tag)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve tag %q in %s: %w", tag, s.layoutPath, err)
	}

	return desc.Digest.String(), tag, nil
}

// CopyPolicy copies a tagged manifest and all its layers from the local OCI
// Layout into the destination store using oras.Copy().
func (s *LocalSource) CopyPolicy(ctx context.Context, _ string, tag string, dst *ocistore.Store) (ocispec.Descriptor, error) {
	srcStore, err := ocistore.New(s.layoutPath)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to open local OCI layout %s: %w", s.layoutPath, err)
	}
	return oras.Copy(ctx, srcStore, tag, dst, tag, oras.CopyOptions{})
}
