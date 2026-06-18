// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"fmt"
	"os"

	ocistore "oras.land/oras-go/v2/content/oci"

	"github.com/complytime/complypack/pkg/complypack"
)

// ComplypackSync provides incremental sync for complypack artifacts.
// Mirrors the Sync/SyncPolicy pattern: compares remote manifest digest
// against local state and skips fetch when unchanged.
type ComplypackSync struct {
	complypackCache *ComplypackCache
	state           *State
	source          ComplypackSource
}

// NewComplypackSync creates a ComplypackSync that orchestrates the
// fetch-unpack-store pipeline for complypack artifacts.
func NewComplypackSync(complypackCache *ComplypackCache, state *State, source ComplypackSource) *ComplypackSync {
	return &ComplypackSync{
		complypackCache: complypackCache,
		state:           state,
		source:          source,
	}
}

// SyncComplypack performs incremental synchronization of a complypack artifact.
// Compares the local cached digest against the remote manifest digest; if they
// match, sync is skipped. On change, the artifact is fetched into a temporary
// OCI Layout store, unpacked via complypack.Unpack(), and stored via
// ComplypackCache.Store(). State is updated and persisted on success.
//
// Returns (true, nil) when a fetch occurred, (false, nil) when the local cache
// was already up-to-date (incremental skip), or (false, err) on failure.
func (s *ComplypackSync) SyncComplypack(ctx context.Context, repository, version string) (bool, error) {
	if repository == "" {
		return false, fmt.Errorf("complypack repository cannot be empty")
	}

	lookupRef := BuildLookupRef(repository, version)

	remoteDigest, remoteVersion, err := s.source.DefinitionVersion(ctx, lookupRef)
	if err != nil {
		return false, fmt.Errorf(
			"complypack %s: registry unreachable: %w (cached data may still be available)",
			repository, err,
		)
	}

	if version == "" || version == "latest" {
		version = remoteVersion
	}

	// Incremental sync check: skip if local digest matches remote.
	//
	// Design note: we only check state digest, not whether the cache directory
	// still exists on disk. The evaluator-id is only known after unpacking the
	// artifact, so we cannot call LookupByEvaluatorID here (we only have the
	// repository string at this point). If a user manually deletes the cache
	// directory but state.json still records a matching digest, this guard will
	// skip the sync. The user must clear state (or change the digest) to force
	// a re-fetch. This matches the policy sync pattern where state is the
	// source of truth for incremental checks.
	localState, exists := s.state.GetComplypackState(repository)
	if exists && localState.Digest == remoteDigest {
		return false, nil
	}

	// Create a temporary OCI Layout store for the oras.Copy() transfer.
	// This is discarded after unpacking — the final cache uses the
	// ComplypackCache directory structure, not an OCI Layout.
	tmpDir, err := os.MkdirTemp("", "complypack-oci-*")
	if err != nil {
		return false, fmt.Errorf("failed to create temporary OCI store directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpStore, err := ocistore.New(tmpDir)
	if err != nil {
		return false, fmt.Errorf("failed to open temporary OCI store: %w", err)
	}

	desc, err := s.source.CopyComplypack(ctx, repository, version, tmpStore)
	if err != nil {
		return false, fmt.Errorf(
			"complypack %s@%s: registry unreachable: %w (local cache unchanged)",
			repository, version, err,
		)
	}

	// Unpack the complypack artifact from the temporary OCI store.
	result, err := complypack.Unpack(ctx, tmpStore, desc)
	if err != nil {
		return false, fmt.Errorf("failed to unpack complypack %s@%s: %w", repository, version, err)
	}
	defer result.Content.Close()

	// Store the unpacked config and content into the ComplypackCache.
	_, err = s.complypackCache.Store(result.Config, result.Content)
	if err != nil {
		return false, fmt.Errorf("failed to store complypack %s@%s: %w", repository, version, err)
	}

	s.state.UpdateComplypackState(repository, version, remoteDigest, result.Config.EvaluatorID)
	if err := SaveState(s.state, s.complypackCache.Dir()); err != nil {
		return false, fmt.Errorf("failed to save state after complypack sync: %w (complypack blobs are valid)", err)
	}

	return true, nil
}
