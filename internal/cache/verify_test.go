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

func TestHexToBytes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{"valid hex", "48656c6c6f", []byte("Hello"), false},
		{"empty string", "", []byte{}, false},
		{"sha256 hex", "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", nil, false},
		{"odd length", "abc", nil, true},
		{"invalid char", "zz", nil, true},
		{"uppercase", "4f4b", []byte("OK"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hexToBytes(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want != nil {
				assert.Equal(t, tt.want, result)
			} else {
				assert.NotEmpty(t, result)
			}
		})
	}
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
