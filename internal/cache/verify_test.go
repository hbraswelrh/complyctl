// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/complytime/complyctl/internal/cache"
)

func TestNoOpVerifier_ReturnsUnverified(t *testing.T) {
	v := cache.NoOpVerifier()

	result, err := v.Verify(context.Background(), ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:abc123",
		Size:      42,
	})

	require.NoError(t, err)
	assert.Equal(t, cache.Unverified, result.Status)
}

func TestNoOpVerifier_NilError(t *testing.T) {
	v := cache.NoOpVerifier()

	_, err := v.Verify(context.Background(), ocispec.Descriptor{})

	require.NoError(t, err)
}

func TestVerificationStatus_Constants(t *testing.T) {
	// Verify the constants have distinct values
	assert.NotEqual(t, cache.Verified, cache.Unverified)
}
