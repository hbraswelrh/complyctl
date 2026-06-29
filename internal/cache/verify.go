// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"fmt"
	"log/slog"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// VerifyFunc checks the cryptographic signature of an OCI artifact.
// Returns (true, nil) when the artifact is verified, (false, nil)
// when unverified, and (false, error) on infrastructure failure.
//
// A nil VerifyFunc means verification is not configured; the artifact
// is treated as unverified.
//
// Design note (complypack#63): When sigstore-go integration lands,
// the verification timing must be decided -- pre-copy (against the
// registry before oras.Copy) vs post-copy (against the local OCI
// store after oras.Copy). Pre-copy prevents caching unverified
// content but requires registry access from the verifier and
// potentially a second OCI client stack (go-containerregistry
// alongside oras-go). Post-copy is simpler and aligns with
// complypack's existing verify() function signature. A --skip-verify
// flag should be added at that time.
type VerifyFunc func(ctx context.Context, desc ocispec.Descriptor) (bool, error)

// SyncResult holds the outcome of a sync operation, indicating
// whether a fetch occurred and whether the artifact was verified.
type SyncResult struct {
	Fetched  bool
	Verified bool
}

// syncOptions holds optional configuration for sync pipelines.
type syncOptions struct {
	verifyFn VerifyFunc
}

// SyncOption configures optional behavior for Sync and ComplypackSync.
type SyncOption func(*syncOptions)

// WithVerifier sets a verification function called after a successful
// artifact copy. When not set, artifacts are treated as unverified.
func WithVerifier(fn VerifyFunc) SyncOption {
	return func(o *syncOptions) {
		o.verifyFn = fn
	}
}

// applySyncOptions processes variadic SyncOption values into a
// syncOptions struct.
func applySyncOptions(opts []SyncOption) syncOptions {
	var o syncOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// runVerify executes the verify function against the copied descriptor.
// Returns true when the artifact is verified, false otherwise.
// A nil verifyFn means verification was not configured (unverified).
// Verifier errors are treated as unverified (graceful degradation)
// and logged as warnings for operational visibility.
func runVerify(ctx context.Context, fn VerifyFunc, desc ocispec.Descriptor) bool {
	if fn == nil {
		return false
	}
	verified, err := fn(ctx, desc)
	if err != nil {
		slog.Warn("Verification failed, treating as unverified",
			"digest", desc.Digest.String(), "error", err)
		return false
	}
	return verified
}

// UnverifiedWarning returns the formatted warning message for an
// unverified artifact. Use this helper in CLI code to avoid
// duplicating warning text across policy and complypack sync paths.
func UnverifiedWarning(artifactType, id string) string {
	return fmt.Sprintf("WARNING: %s %s has not been cryptographically verified", artifactType, id)
}
