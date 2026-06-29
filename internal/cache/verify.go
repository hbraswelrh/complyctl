// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// VerificationStatus represents the outcome of a cryptographic
// verification attempt on an OCI artifact.
type VerificationStatus int

const (
	// Unverified indicates the artifact has not been
	// cryptographically verified.
	Unverified VerificationStatus = iota
	// Verified indicates the artifact has been
	// cryptographically verified.
	Verified
)

// VerificationResult holds the outcome of a verification attempt.
type VerificationResult struct {
	Status VerificationStatus
}

// Verifier checks the cryptographic signature of an OCI artifact.
// Implementations receive the descriptor returned by CopyPolicy and
// return whether the artifact is verified.
//
// When the upstream sigstore-go integration (complytime/complypack#63)
// lands, a real implementation will replace NoOpVerifier.
type Verifier interface {
	Verify(ctx context.Context, desc ocispec.Descriptor) (VerificationResult, error)
}

// noOpVerifier is the default Verifier that always returns Unverified
// with a nil error. It serves as a placeholder until signature
// verification is implemented.
type noOpVerifier struct{}

// NoOpVerifier returns a Verifier that always reports artifacts as
// unverified. Use as the default when no signature verification is
// configured.
func NoOpVerifier() Verifier {
	return &noOpVerifier{}
}

func (v *noOpVerifier) Verify(_ context.Context, _ ocispec.Descriptor) (VerificationResult, error) {
	return VerificationResult{Status: Unverified}, nil
}
