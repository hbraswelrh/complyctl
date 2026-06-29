// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/complytime/complyctl/internal/registry"
)

// BuildLookupRef constructs an OCI lookup reference from a repository with
// separate tag and digest fields. When digest is non-empty it uses "@" as
// the separator; when tag is non-empty (and not "latest") it uses ":".
// If both are empty or tag is "latest", the bare repository is returned so
// oras resolves the default tag.
func BuildLookupRef(repository, tag, digest string) string {
	if digest != "" {
		return repository + "@" + digest
	}
	if tag == "" || tag == "latest" {
		return repository
	}
	return repository + ":" + tag
}

// classifyVersion splits a version string into tag and digest components.
// It delegates to go-digest's Parse to determine whether the string is a
// valid OCI digest (sha256, sha384, sha512). This is a convenience for
// callers that receive an untyped version string and need to call
// BuildLookupRef.
func classifyVersion(version string) (tag string, dgst string) {
	if _, err := digest.Parse(version); err == nil {
		return "", version
	}
	return version, ""
}

// Sync provides incremental sync using oras.Copy() for remote-to-local transfer.
type Sync struct {
	cache    *Cache
	state    *State
	source   PolicySource
	verifier Verifier
}

// NewSync creates a Sync instance with the given cache, state, source,
// and verifier. The verifier is called after each successful CopyPolicy
// to determine whether the fetched artifact is cryptographically
// verified. Pass NoOpVerifier() when no verification is configured.
func NewSync(cache *Cache, state *State, source PolicySource, verifier Verifier) *Sync {
	return &Sync{
		cache:    cache,
		state:    state,
		source:   source,
		verifier: verifier,
	}
}

// SyncPolicy performs incremental synchronization of a policy. Compares local
// digest against remote manifest digest; if they match, sync is skipped. On
// failure, the OCI Layout store retains its previous state.
//
// Returns (true, nil) when a fetch occurred, (false, nil) when the local cache
// was already up-to-date (incremental skip), or (false, err) on failure.
func (s *Sync) SyncPolicy(ctx context.Context, policyID, version string) (bool, error) {
	if policyID == "" {
		return false, fmt.Errorf("policy ID cannot be empty")
	}

	tag, digest := classifyVersion(version)
	lookupRef := BuildLookupRef(policyID, tag, digest)

	remoteDigest, remoteVersion, err := s.source.DefinitionVersion(ctx, lookupRef)
	if err != nil {
		if errors.Is(err, registry.ErrVersionNotFound) {
			return false, fmt.Errorf("policy %s: %w", policyID, err)
		}
		return false, fmt.Errorf("policy %s: registry unreachable: %w", policyID, err)
	}

	if version == "" || version == "latest" {
		version = remoteVersion
	}

	localState, exists := s.state.GetPolicyState(policyID)
	if exists && localState.Digest == remoteDigest && s.cache.PolicyStoreExists(policyID) {
		return false, nil
	}

	localStore, err := s.cache.NewPolicyStore(policyID)
	if err != nil {
		return false, fmt.Errorf("failed to open local store for policy %s: %w", policyID, err)
	}

	desc, err := s.source.CopyPolicy(ctx, policyID, version, localStore)
	if err != nil {
		return false, fmt.Errorf("policy %s@%s: copy failed: %w", policyID, version, err)
	}

	verified := verifyAfterCopy(ctx, s.verifier, desc)

	s.state.UpdatePolicyState(policyID, version, remoteDigest, verified)
	if err := SaveState(s.state, s.cache.Dir()); err != nil {
		return false, fmt.Errorf("failed to save state after sync: %w (policy blobs are valid)", err)
	}

	return true, nil
}

// verifyAfterCopy runs the verifier against the copied descriptor.
// Returns true when the artifact is verified, false otherwise.
// Verifier errors are treated as unverified (graceful degradation).
func verifyAfterCopy(ctx context.Context, v Verifier, desc ocispec.Descriptor) bool {
	result, err := v.Verify(ctx, desc)
	if err != nil {
		return false
	}
	return result.Status == Verified
}
