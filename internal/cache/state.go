// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/complytime/complyctl/internal/complytime"
)

// State tracks sync metadata for all cached policies and complypacks,
// persisted as state.json.
type State struct {
	LastSync    time.Time              `json:"last_sync"`
	Policies    map[string]PolicyState `json:"policies"`
	Complypacks map[string]PolicyState `json:"complypacks,omitempty"`
}

// PolicyState holds version, digest, verification status, and timestamp for
// a single cached policy or complypack.
type PolicyState struct {
	Version        string    `json:"version"`
	Digest         string    `json:"digest"`
	EvaluatorID    string    `json:"evaluator_id,omitempty"`
	LastUpdated    time.Time `json:"last_updated"`
	Verified       bool      `json:"verified,omitempty"`
	SignerIdentity string    `json:"signer_identity,omitempty"`
	Issuer         string    `json:"issuer,omitempty"`
	VerifiedAt     time.Time `json:"verified_at,omitempty"`
}

// LoadState reads and parses the state.json file from the given cache directory.
// Returns a fresh State with empty maps if the file does not exist.
func LoadState(cacheDir string) (*State, error) {
	statePath := filepath.Join(cacheDir, complytime.StateFileName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{
				LastSync:    time.Time{},
				Policies:    make(map[string]PolicyState),
				Complypacks: make(map[string]PolicyState),
			}, nil
		}
		return nil, fmt.Errorf("failed to read state file %s: %w", statePath, err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file %s: %w", statePath, err)
	}

	initStateMaps(&state)

	return &state, nil
}

// initStateMaps ensures Policies and Complypacks maps are non-nil.
// Extracted to keep LoadState's cyclomatic complexity stable when new
// map fields are added to State.
func initStateMaps(s *State) {
	if s.Policies == nil {
		s.Policies = make(map[string]PolicyState)
	}
	if s.Complypacks == nil {
		s.Complypacks = make(map[string]PolicyState)
	}
}

// SaveState atomically writes the state to state.json in the given cache
// directory. It marshals to a sibling temp file then renames it into place so
// concurrent readers never observe a truncated or partial write.
func SaveState(state *State, cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	statePath := filepath.Join(cacheDir, complytime.StateFileName)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to a temp file in the same directory so os.Rename is atomic
	// (same filesystem, POSIX guarantee). This prevents concurrent readers
	// from observing a truncated file mid-write.
	tmp, err := os.CreateTemp(cacheDir, ".state-*.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp state file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp state file %s: %w", tmpPath, err)
	}
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to set permissions on temp state file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp state file: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write state file %s: %w", statePath, err)
	}

	return nil
}

// UpdatePolicyStateWithVerification records version, digest, verification
// metadata, and current timestamp for a cached policy. When vr is nil,
// Verified is set to false (no verification was performed).
func (s *State) UpdatePolicyStateWithVerification(policyID, version, digest string, vr *VerificationResult) {
	if s.Policies == nil {
		s.Policies = make(map[string]PolicyState)
	}
	ps := PolicyState{
		Version:     version,
		Digest:      digest,
		LastUpdated: time.Now(),
	}
	if vr != nil {
		ps.Verified = vr.Verified
		ps.SignerIdentity = vr.SignerIdentity
		ps.Issuer = vr.Issuer
		ps.VerifiedAt = vr.VerifiedAt
	}
	s.Policies[policyID] = ps
	s.LastSync = time.Now()
}

// UpdateComplypackStateWithVerification records version, digest, evaluator-id,
// verification metadata, and current timestamp for a cached complypack.
func (s *State) UpdateComplypackStateWithVerification(repository, version, digest, evaluatorID string, vr *VerificationResult) {
	if s.Complypacks == nil {
		s.Complypacks = make(map[string]PolicyState)
	}
	ps := PolicyState{
		Version:     version,
		Digest:      digest,
		EvaluatorID: evaluatorID,
		LastUpdated: time.Now(),
	}
	if vr != nil {
		ps.Verified = vr.Verified
		ps.SignerIdentity = vr.SignerIdentity
		ps.Issuer = vr.Issuer
		ps.VerifiedAt = vr.VerifiedAt
	}
	s.Complypacks[repository] = ps
	s.LastSync = time.Now()
}

// GetPolicyState returns the cached state for a policy identified by policyID.
func (s *State) GetPolicyState(policyID string) (PolicyState, bool) {
	if s.Policies == nil {
		return PolicyState{}, false
	}
	state, exists := s.Policies[policyID]
	return state, exists
}

// GetComplypackState returns the cached state for a complypack, keyed by
// repository (e.g., "example.com/complypacks/opa-bundle").
func (s *State) GetComplypackState(repository string) (PolicyState, bool) {
	if s.Complypacks == nil {
		return PolicyState{}, false
	}
	state, exists := s.Complypacks[repository]
	return state, exists
}
