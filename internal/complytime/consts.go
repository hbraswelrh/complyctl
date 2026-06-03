// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const stateSubdir = ".complytime"
const providerSubdir = "providers"

// WorkspaceDir is the workspace-local directory for all complyctl artifacts
// (generation state, scan output). Separate from the global ~/.complytime/ cache.
const WorkspaceDir = ".complytime"

const StateFileName = "state.json"
const PoliciesSubdir = "policies"
const ComplypacksSubdir = "complypacks"
const WorkspaceConfigFile = "complytime.yaml"

const CurrentWorkspaceVersion = 1

const (
	OutputFormatOSCAL  = "oscal"
	OutputFormatPretty = "pretty"
	OutputFormatSARIF  = "sarif"
)

// ExportEnabledEnvVar is the environment variable that controls whether
// the scan command triggers evidence export to a Beacon collector.
// Parsed via strconv.ParseBool — accepts "true", "TRUE", "1", etc.
const ExportEnabledEnvVar = "COMPLYTIME_EXPORT_ENABLED"

// ExportEnabled checks whether COMPLYTIME_EXPORT_ENABLED is set to a truthy
// value. Returns (true, "", nil) when enabled, (false, "", nil) when unset or
// falsy, and (false, rawValue, err) when set to an unrecognized value.
func ExportEnabled() (enabled bool, raw string, err error) {
	raw = os.Getenv(ExportEnabledEnvVar)
	if raw == "" {
		return false, "", nil
	}
	enabled, err = strconv.ParseBool(raw)
	if err != nil {
		return false, raw, err
	}
	return enabled, raw, nil
}

const ScanOutputDir = "scan"

// LogFileName is the log file name written to {WorkspaceDir}/{LogFileName}.
// See FR-038, R57: specs/001-gemara-native-workflow/research.md
const LogFileName = "complyctl.log"

// DefaultCommandTimeout is the default deadline for scan and generate operations.
// This flows through gRPC to the provider subprocess without additional capping.
const DefaultCommandTimeout = 5 * time.Minute

const ProviderExecutablePrefix = "complyctl-provider-"

// SystemProviderDir is the system-wide provider directory where
// package managers (e.g., RPM) install provider binaries.
// Discovery checks this path as a fallback after the user directory.
const SystemProviderDir = "/usr/libexec/complytime/providers"

// Gemara OCI layer media types for identifying layer content within multi-layer OCI manifests.
const (
	MediaTypeCatalog  = "application/vnd.gemara.catalog.v1+yaml"
	MediaTypeGuidance = "application/vnd.gemara.guidance.v1+yaml"
	MediaTypePolicy   = "application/vnd.gemara.policy.v1+yaml"
)

const OCIEmptyConfig = "application/vnd.oci.empty.v1+json"

// Scan result status emoji indicators for terminal summary table (FR-037).
const (
	StatusPassed  = "✅"
	StatusFailed  = "❌"
	StatusSkipped = "⏭️"
	StatusError   = "⚠️"
)

// FilenameSafe replaces characters unsafe for filenames (e.g., path separators)
// so that policy IDs like "policies/nist-800-53-r5" produce flat filenames.
func FilenameSafe(s string) string {
	return strings.ReplaceAll(s, "/", "-")
}

// ExpandPath resolves a leading ~/ to the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// ResolveCacheDir returns the absolute path to the cache directory.
func ResolveCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, stateSubdir), nil
}

// ResolveProviderDir returns the absolute path to the provider directory.
func ResolveProviderDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, stateSubdir, providerSubdir), nil
}
