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

// PolicyState holds version, digest, and timestamp for a single cached policy.
type PolicyState struct {
	Version     string    `json:"version"`
	Digest      string    `json:"digest"`
	EvaluatorID string    `json:"evaluator_id,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
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

// SaveState writes the state to state.json in the given cache directory.
func SaveState(state *State, cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	statePath := filepath.Join(cacheDir, complytime.StateFileName)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file %s: %w", statePath, err)
	}

	return nil
}

// UpdatePolicyState records the version, digest, and current timestamp for a cached policy.
func (s *State) UpdatePolicyState(policyID, version, digest string) {
	if s.Policies == nil {
		s.Policies = make(map[string]PolicyState)
	}
	s.Policies[policyID] = PolicyState{
		Version:     version,
		Digest:      digest,
		LastUpdated: time.Now(),
	}
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

// UpdateComplypackState records the version, digest, evaluator-id, and current
// timestamp for a cached complypack, keyed by repository
// (e.g., "example.com/complypacks/opa-bundle").
func (s *State) UpdateComplypackState(repository, version, digest, evaluatorID string) {
	if s.Complypacks == nil {
		s.Complypacks = make(map[string]PolicyState)
	}
	s.Complypacks[repository] = PolicyState{
		Version:     version,
		Digest:      digest,
		EvaluatorID: evaluatorID,
		LastUpdated: time.Now(),
	}
	s.LastSync = time.Now()
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
