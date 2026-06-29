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
	verifyFn        VerifyFunc
}

// NewComplypackSync creates a ComplypackSync that orchestrates the
// fetch-unpack-store pipeline for complypack artifacts. Pass
// WithVerifier to configure cryptographic verification of fetched
// artifacts. When no verifier is set, artifacts are treated as
// unverified.
func NewComplypackSync(complypackCache *ComplypackCache, state *State, source ComplypackSource, opts ...SyncOption) *ComplypackSync {
	o := applySyncOptions(opts)
	return &ComplypackSync{
		complypackCache: complypackCache,
		state:           state,
		source:          source,
		verifyFn:        o.verifyFn,
	}
}

// SyncComplypack performs incremental synchronization of a complypack artifact.
// Compares the local cached digest against the remote manifest digest; if they
// match, sync is skipped. On change, the artifact is fetched into a temporary
// OCI Layout store, unpacked via complypack.Unpack(), and stored via
// ComplypackCache.Store(). State is updated and persisted on success.
//
// Returns a SyncResult indicating whether a fetch occurred and whether the
// artifact was verified. Returns an error on sync failure.
func (s *ComplypackSync) SyncComplypack(ctx context.Context, repository, version string) (SyncResult, error) {
	if repository == "" {
		return SyncResult{}, fmt.Errorf("complypack repository cannot be empty")
	}

	tag, digest := classifyVersion(version)
	lookupRef := BuildLookupRef(repository, tag, digest)

	remoteDigest, remoteVersion, err := s.source.DefinitionVersion(ctx, lookupRef)
	if err != nil {
		return SyncResult{}, fmt.Errorf(
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
		return SyncResult{}, nil
	}

	// Create a temporary OCI Layout store for the oras.Copy() transfer.
	// This is discarded after unpacking — the final cache uses the
	// ComplypackCache directory structure, not an OCI Layout.
	tmpDir, err := os.MkdirTemp("", "complypack-oci-*")
	if err != nil {
		return SyncResult{}, fmt.Errorf("failed to create temporary OCI store directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpStore, err := ocistore.New(tmpDir)
	if err != nil {
		return SyncResult{}, fmt.Errorf("failed to open temporary OCI store: %w", err)
	}

	desc, err := s.source.CopyComplypack(ctx, repository, version, tmpStore)
	if err != nil {
		return SyncResult{}, fmt.Errorf(
			"complypack %s@%s: registry unreachable: %w (local cache unchanged)",
			repository, version, err,
		)
	}

	verified := runVerify(ctx, s.verifyFn, desc)

	// Unpack the complypack artifact from the temporary OCI store.
	unpackResult, err := complypack.Unpack(ctx, tmpStore, desc)
	if err != nil {
		return SyncResult{}, fmt.Errorf("failed to unpack complypack %s@%s: %w", repository, version, err)
	}
	defer unpackResult.Content.Close()

	// Store the unpacked config and content into the ComplypackCache.
	_, err = s.complypackCache.Store(unpackResult.Config, unpackResult.Content)
	if err != nil {
		return SyncResult{}, fmt.Errorf("failed to store complypack %s@%s: %w", repository, version, err)
	}

	s.state.UpdateComplypackState(repository, version, remoteDigest, unpackResult.Config.EvaluatorID, verified)
	if err := SaveState(s.state, s.complypackCache.Dir()); err != nil {
		return SyncResult{}, fmt.Errorf("failed to save state after complypack sync: %w (complypack blobs are valid)", err)
	}

	return SyncResult{Fetched: true, Verified: verified}, nil
}
