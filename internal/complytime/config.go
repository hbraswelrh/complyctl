// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// unsafeRefChars matches characters that should never appear in an OCI reference
// and are commonly used for shell injection.
var unsafeRefChars = regexp.MustCompile("[;|&$`!><(){}\\[\\]\\\\]")

// ValidateOCIRef checks that raw looks like a valid OCI reference
// (registry/repository with optional :tag or @version). It rejects empty
// strings, shell metacharacters, and bare names without a registry component.
func ValidateOCIRef(raw string) error {
	s := strings.TrimSpace(raw)
	if s == "" {
		return fmt.Errorf("policy URL cannot be empty")
	}
	if unsafeRefChars.MatchString(s) {
		return fmt.Errorf("policy URL contains invalid characters: %s", s)
	}

	stripped := strings.TrimPrefix(strings.TrimPrefix(s, "https://"), "http://")
	if !strings.Contains(stripped, "/") {
		return fmt.Errorf("policy URL must include a registry and repository (e.g. registry.com/repo:tag): %s", s)
	}

	host := strings.SplitN(stripped, "/", 2)[0]
	if !strings.Contains(host, ".") && !strings.Contains(host, ":") {
		return fmt.Errorf("policy URL must include a registry host (e.g. registry.com/repo:tag): %s", s)
	}

	return nil
}

// WorkspaceConfig is the top-level YAML configuration for a complytime workspace.
// See R48, R49: three-tier variable model.
//
// Variables holds workspace-scoped constants (e.g., output directory paths).
// These are NOT expanded for ${...} environment references — only
// target-level variables support ${VAR} substitution (for secrets and
// per-target credentials). Place environment-dependent values in
// targets[].variables instead.
type WorkspaceConfig struct {
	Version     int               `yaml:"version,omitempty"`
	Policies    []PolicyEntry     `yaml:"policies"`
	Complypacks []PolicyEntry     `yaml:"complypacks,omitempty"`
	Targets     []TargetConfig    `yaml:"targets"`
	Variables   map[string]string `yaml:"variables,omitempty"`
	Collector   *CollectorConfig  `yaml:"collector,omitempty"`
}

// CollectorConfig holds the Beacon collector endpoint and OIDC credentials.
type CollectorConfig struct {
	Endpoint string      `yaml:"endpoint"`
	Auth     *AuthConfig `yaml:"auth,omitempty"`
}

// AuthConfig holds OIDC client credentials for collector authentication.
type AuthConfig struct {
	ClientID      string `yaml:"client-id"`
	ClientSecret  string `yaml:"client-secret"` //nolint:gosec // not a hardcoded credential
	TokenEndpoint string `yaml:"token-endpoint"`
}

// PolicyEntry pairs a full OCI reference with an optional user-chosen shortname.
// If ID is empty, EffectiveID derives one from the last path segment of URL.
type PolicyEntry struct {
	URL string `yaml:"url"`
	ID  string `yaml:"id,omitempty"`
}

// EffectiveID returns the explicit ID if set, otherwise derives one from the
// last path segment of the URL (e.g. "registry.com/policies/nist-r5@v1" → "nist-r5").
func (p PolicyEntry) EffectiveID() string {
	if p.ID != "" {
		return p.ID
	}
	ref := ParsePolicyRef(p.URL)
	segments := strings.Split(ref.Repository, "/")
	return segments[len(segments)-1]
}

// TargetConfig binds a scan target to one or more policies with optional variables.
// Policies are referenced by their effective ID (explicit or derived).
type TargetConfig struct {
	ID        string            `yaml:"id"`
	Policies  []string          `yaml:"policies"`
	Variables map[string]string `yaml:"variables,omitempty"`
}

// PolicyRef represents a parsed OCI policy reference.
type PolicyRef struct {
	Raw        string
	Registry   string
	Repository string
	Version    string
}

// ParsePolicyRef parses a full OCI reference into its components.
// Handles optional scheme (http://, https://), registry host detection,
// and @version suffix.
func ParsePolicyRef(raw string) PolicyRef {
	ref := PolicyRef{Raw: raw}
	s := strings.TrimSpace(raw)

	var scheme string
	if strings.HasPrefix(s, "http://") {
		scheme = "http://"
		s = strings.TrimPrefix(s, "http://")
	} else if strings.HasPrefix(s, "https://") {
		scheme = "https://"
		s = strings.TrimPrefix(s, "https://")
	}

	if idx := strings.LastIndex(s, "@"); idx > 0 && idx < len(s)-1 {
		ref.Version = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 2 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		ref.Registry = scheme + parts[0]
		ref.Repository = parts[1]
	} else {
		ref.Repository = s
	}

	return ref
}

// FindPolicy matches a policy identifier against the policies list by effective ID.
func FindPolicy(policies []PolicyEntry, policyID string) (*PolicyEntry, bool) {
	policyID = strings.TrimSpace(policyID)

	for i, p := range policies {
		if p.EffectiveID() == policyID {
			return &policies[i], true
		}
	}

	return nil, false
}

// PolicyIDs returns the effective IDs for all policies in the config.
func PolicyIDs(policies []PolicyEntry) map[string]*PolicyEntry {
	m := make(map[string]*PolicyEntry, len(policies))
	for i := range policies {
		m[policies[i].EffectiveID()] = &policies[i]
	}
	return m
}

// LoadFrom reads, parses, and validates the complytime configuration from the
// given path.
func LoadFrom(configPath string) (*WorkspaceConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"complytime config not found: %s (run 'complyctl init' to create)",
				configPath,
			)
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config WorkspaceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf(
			"corrupted complytime file %s: invalid YAML: %w",
			configPath, err,
		)
	}

	if err := resolveEnvVars(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// resolveEnvVars expands ${VAR} references in target variable values
// and collector auth fields from the process environment. Returns an
// error if a referenced variable is not set.
func resolveEnvVars(config *WorkspaceConfig) error {
	for i, target := range config.Targets {
		for key, val := range target.Variables {
			resolved, err := expandEnvRef(val)
			if err != nil {
				return fmt.Errorf("targets[%s].variables.%s: %w", target.ID, key, err)
			}
			config.Targets[i].Variables[key] = resolved
		}
	}
	return resolveCollectorEnvVars(config.Collector)
}

func resolveCollectorEnvVars(collector *CollectorConfig) error {
	if collector == nil || collector.Auth == nil {
		return nil
	}
	auth := collector.Auth
	fields := map[string]*string{
		"collector.auth.client-id":      &auth.ClientID,
		"collector.auth.client-secret":  &auth.ClientSecret,
		"collector.auth.token-endpoint": &auth.TokenEndpoint,
	}
	for label, ptr := range fields {
		resolved, err := expandEnvRef(*ptr)
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		*ptr = resolved
	}
	return nil
}

// expandEnvRef replaces all ${VAR} occurrences in s with their environment
// values. Returns an error if any referenced variable is unset.
func expandEnvRef(s string) (string, error) {
	var missing []string
	result := envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		missing = append(missing, varName)
		return match
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("unset environment variable(s): %s", strings.Join(missing, ", "))
	}
	return result, nil
}

// SaveTo writes complytime configuration to the given path.
func SaveTo(config *WorkspaceConfig, configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}

	return nil
}

// Validate checks required fields, uniqueness constraints, and policy format.
func Validate(config *WorkspaceConfig) error {
	if config.Version != 0 && config.Version != CurrentWorkspaceVersion {
		return fmt.Errorf(
			"unsupported config version %d (expected %d)",
			config.Version, CurrentWorkspaceVersion,
		)
	}

	if len(config.Policies) == 0 {
		return fmt.Errorf("policies: at least one policy is required")
	}

	if err := validateEntries("policies", config.Policies); err != nil {
		return err
	}
	if err := validateEntries("complypacks", config.Complypacks); err != nil {
		return err
	}

	policyLookup := PolicyIDs(config.Policies)

	targetIDs := make(map[string]bool)
	for _, target := range config.Targets {
		if target.ID == "" {
			return fmt.Errorf("targets[].id cannot be empty")
		}
		if targetIDs[target.ID] {
			return fmt.Errorf("targets[].id: duplicate %s", target.ID)
		}
		targetIDs[target.ID] = true
		if len(target.Policies) == 0 {
			return fmt.Errorf("targets[%s].policies: at least one required", target.ID)
		}
		seenTargetPolicies := make(map[string]bool)
		for _, pid := range target.Policies {
			if _, ok := policyLookup[pid]; !ok {
				return fmt.Errorf("targets[%s]: policy %q not in policies list", target.ID, pid)
			}
			if seenTargetPolicies[pid] {
				return fmt.Errorf("targets[%s]: duplicate policy %s", target.ID, pid)
			}
			seenTargetPolicies[pid] = true
		}
	}
	return nil
}

// validateEntries checks uniqueness and OCI reference validity for a
// list of policy or complypack entries. An empty or nil list is valid.
// The label parameter (e.g., "policies", "complypacks") is used in
// error messages. Extracted from Validate to keep its cyclomatic
// complexity stable as new entry lists are added.
func validateEntries(label string, entries []PolicyEntry) error {
	if len(entries) == 0 {
		return nil
	}

	seenURL := make(map[string]bool)
	seenID := make(map[string]bool)
	for _, entry := range entries {
		if err := ValidateOCIRef(entry.URL); err != nil {
			return fmt.Errorf("%s[]: %w", label, err)
		}
		if seenURL[entry.URL] {
			return fmt.Errorf("%s: duplicate url %s", label, entry.URL)
		}
		seenURL[entry.URL] = true

		eid := entry.EffectiveID()
		if seenID[eid] {
			return fmt.Errorf("%s: duplicate id %s", label, eid)
		}
		seenID[eid] = true
	}
	return nil
}
