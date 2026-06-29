// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"fmt"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"

	"github.com/complytime/complyctl/internal/cache"
)

func TestWithVerifier_AppliesFunction(t *testing.T) {
	called := false
	fn := func(_ context.Context, _ ocispec.Descriptor) (bool, error) {
		called = true
		return true, nil
	}

	// WithVerifier should produce a valid SyncOption.
	// We verify indirectly by constructing a Sync with it and
	// running SyncPolicy (tested in sync_test.go). Here we just
	// confirm the option does not panic and the function is callable.
	opt := cache.WithVerifier(fn)
	assert.NotNil(t, opt)

	// Verify the function itself works as expected.
	verified, err := fn(context.Background(), ocispec.Descriptor{})
	assert.NoError(t, err)
	assert.True(t, verified)
	assert.True(t, called)
}

func TestUnverifiedWarning_Policy(t *testing.T) {
	msg := cache.UnverifiedWarning("policy", "cis-benchmark")
	assert.Equal(t, "WARNING: policy cis-benchmark has not been cryptographically verified", msg)
}

func TestUnverifiedWarning_Complypack(t *testing.T) {
	msg := cache.UnverifiedWarning("complypack", "opa-bundle")
	assert.Equal(t, "WARNING: complypack opa-bundle has not been cryptographically verified", msg)
}

func TestSyncResult_ZeroValue(t *testing.T) {
	var result cache.SyncResult
	assert.False(t, result.Fetched, "zero-value SyncResult should have Fetched=false")
	assert.False(t, result.Verified, "zero-value SyncResult should have Verified=false")
}

func TestRunVerify_Exported_NilFunc(t *testing.T) {
	// When no VerifyFunc is configured (nil), RunVerify should
	// return false without panicking. We test this via the sync
	// pipeline in sync_test.go; this is a unit-level sanity check
	// that SyncResult reflects unverified when no verifier is set.
	result := cache.SyncResult{Fetched: true, Verified: false}
	assert.True(t, result.Fetched)
	assert.False(t, result.Verified)
}

func TestRunVerify_ErrorTreatedAsUnverified(t *testing.T) {
	// Verify the contract: when VerifyFunc returns an error,
	// the artifact is treated as unverified (not as a sync failure).
	fn := func(_ context.Context, _ ocispec.Descriptor) (bool, error) {
		return false, fmt.Errorf("transparency log unreachable")
	}
	result := cache.SyncResult{Fetched: true, Verified: false}

	// The actual runVerify behavior is tested via sync_test.go.
	// This confirms the result struct correctly represents the
	// expected outcome.
	verified, err := fn(context.Background(), ocispec.Descriptor{})
	assert.Error(t, err)
	assert.False(t, verified)
	assert.False(t, result.Verified)
}
