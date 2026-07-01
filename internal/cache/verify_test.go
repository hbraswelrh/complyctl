// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyFunc_NilSkipsVerification(t *testing.T) {
	var vf VerifyFunc
	assert.Nil(t, vf, "nil VerifyFunc represents disabled verification")
}

func TestVerifyFunc_MockSuccess(t *testing.T) {
	mockVerifier := func(_ context.Context, ref string) (*VerificationResult, error) {
		return &VerificationResult{
			Verified:       true,
			SignerIdentity: "test@example.com",
			Issuer:         "https://issuer.example.com",
			VerifiedAt:     time.Now(),
		}, nil
	}

	result, err := mockVerifier(context.Background(), "registry.com/repo:v1.0")
	require.NoError(t, err)
	assert.True(t, result.Verified)
	assert.Equal(t, "test@example.com", result.SignerIdentity)
	assert.Equal(t, "https://issuer.example.com", result.Issuer)
	assert.False(t, result.VerifiedAt.IsZero())
}

func TestVerifyFunc_MockFailure(t *testing.T) {
	mockVerifier := func(_ context.Context, ref string) (*VerificationResult, error) {
		return nil, fmt.Errorf("signature verification failed for %s: identity mismatch", ref)
	}

	result, err := mockVerifier(context.Background(), "registry.com/repo:v1.0")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "identity mismatch")
}

func TestParseCertificateChain_InvalidPEM(t *testing.T) {
	_, err := parseCertificateChain("not a PEM")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no certificates found")
}

func TestParseCertificateChain_EmptyPEM(t *testing.T) {
	_, err := parseCertificateChain("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no certificates found")
}

func TestParseRekorBundle_ValidJSON(t *testing.T) {
	bodyB64 := base64.StdEncoding.EncodeToString([]byte(`{"test": "body"}`))
	setB64 := base64.StdEncoding.EncodeToString([]byte("signed-timestamp"))
	logIDB64 := base64.StdEncoding.EncodeToString([]byte("log-id-bytes"))

	payload := rekorBundlePayload{
		SignedEntryTimestamp: setB64,
	}
	payload.Payload.Body = bodyB64
	payload.Payload.IntegratedTime = 1701205628
	payload.Payload.LogIndex = 12345
	payload.Payload.LogID = logIDB64

	jsonBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	entries, err := parseRekorBundle(string(jsonBytes))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, int64(12345), entries[0].LogIndex)
	assert.Equal(t, int64(1701205628), entries[0].IntegratedTime)
	assert.Equal(t, "hashedrekord", entries[0].KindVersion.Kind)
	assert.NotNil(t, entries[0].InclusionPromise)
}

func TestParseRekorBundle_InvalidJSON(t *testing.T) {
	_, err := parseRekorBundle("not json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestBuildProtobufBundle_MissingAnnotations(t *testing.T) {
	layer := &v1.Descriptor{
		MediaType: types.MediaType(cosignSimpleSigningMediaType),
	}
	_, err := buildProtobufBundle(layer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no annotations")
}

func TestBuildProtobufBundle_MissingSignature(t *testing.T) {
	// Annotations present but no signature annotation — should fail
	layer := &v1.Descriptor{
		MediaType: types.MediaType(cosignSimpleSigningMediaType),
		Annotations: map[string]string{
			"some-annotation": "some-value",
		},
	}
	_, err := buildProtobufBundle(layer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing signature annotation")
}

func TestVerificationResult_ZeroValue(t *testing.T) {
	vr := &VerificationResult{}
	assert.False(t, vr.Verified)
	assert.Empty(t, vr.SignerIdentity)
	assert.Empty(t, vr.Issuer)
	assert.True(t, vr.VerifiedAt.IsZero())
}

func TestPolicyState_BackwardCompatibility(t *testing.T) {
	// Old-format JSON without verification fields must deserialize correctly
	oldJSON := `{"version":"v1.0","digest":"sha256:abc123","last_updated":"2024-01-01T00:00:00Z"}`
	var ps PolicyState
	err := json.Unmarshal([]byte(oldJSON), &ps)
	require.NoError(t, err)
	assert.Equal(t, "v1.0", ps.Version)
	assert.Equal(t, "sha256:abc123", ps.Digest)
	assert.False(t, ps.Verified)
	assert.Empty(t, ps.SignerIdentity)
	assert.Empty(t, ps.Issuer)
	assert.True(t, ps.VerifiedAt.IsZero())
}

func TestPolicyState_OmitemptyMarshal(t *testing.T) {
	// Unverified state should not emit boolean/string verification fields
	// (omitempty works on bool and string). time.Time zero value is always
	// emitted since omitempty does not apply to structs in encoding/json.
	ps := PolicyState{
		Version:     "v1.0",
		Digest:      "sha256:abc123",
		LastUpdated: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	data, err := json.Marshal(ps)
	require.NoError(t, err)
	s := string(data)
	assert.NotContains(t, s, `"verified":`)
	assert.NotContains(t, s, "signer_identity")
	assert.NotContains(t, s, "issuer")
}

func TestPolicyState_VerifiedMarshal(t *testing.T) {
	verifiedAt := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	ps := PolicyState{
		Version:        "v1.0",
		Digest:         "sha256:abc123",
		LastUpdated:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Verified:       true,
		SignerIdentity: "workflow@github.com",
		Issuer:         "https://token.actions.githubusercontent.com",
		VerifiedAt:     verifiedAt,
	}
	data, err := json.Marshal(ps)
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, `"verified":true`)
	assert.Contains(t, s, `"signer_identity":"workflow@github.com"`)
	assert.Contains(t, s, `"issuer":"https://token.actions.githubusercontent.com"`)
	assert.Contains(t, s, `"verified_at"`)
}

func TestUpdatePolicyStateWithVerification_NilResult(t *testing.T) {
	state := &State{Policies: make(map[string]PolicyState)}
	state.UpdatePolicyStateWithVerification("test-policy", "v1.0", "sha256:abc", nil)
	ps, ok := state.GetPolicyState("test-policy")
	require.True(t, ok)
	assert.Equal(t, "v1.0", ps.Version)
	assert.Equal(t, "sha256:abc", ps.Digest)
	assert.False(t, ps.Verified)
	assert.Empty(t, ps.SignerIdentity)
}

func TestUpdatePolicyStateWithVerification_WithResult(t *testing.T) {
	state := &State{Policies: make(map[string]PolicyState)}
	vr := &VerificationResult{
		Verified:       true,
		SignerIdentity: "user@example.com",
		Issuer:         "https://issuer.example.com",
		VerifiedAt:     time.Now(),
	}
	state.UpdatePolicyStateWithVerification("test-policy", "v1.0", "sha256:abc", vr)
	ps, ok := state.GetPolicyState("test-policy")
	require.True(t, ok)
	assert.True(t, ps.Verified)
	assert.Equal(t, "user@example.com", ps.SignerIdentity)
	assert.Equal(t, "https://issuer.example.com", ps.Issuer)
	assert.False(t, ps.VerifiedAt.IsZero())
}

func TestUpdateComplypackStateWithVerification_WithResult(t *testing.T) {
	state := &State{Complypacks: make(map[string]PolicyState)}
	vr := &VerificationResult{
		Verified:       true,
		SignerIdentity: "build@ci.com",
		Issuer:         "https://ci.issuer.com",
		VerifiedAt:     time.Now(),
	}
	state.UpdateComplypackStateWithVerification("repo/pack", "v2.0", "sha256:def", "opa", vr)
	ps, ok := state.GetComplypackState("repo/pack")
	require.True(t, ok)
	assert.True(t, ps.Verified)
	assert.Equal(t, "build@ci.com", ps.SignerIdentity)
	assert.Equal(t, "opa", ps.EvaluatorID)
}
