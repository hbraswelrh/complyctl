## Why

Private policy bundles cannot be committed to GitHub or pushed to a public OCI registry. Today `complyctl get` only supports syncing from remote OCI registries (or a local mock registry that requires content embedded at compile time). Users evaluating private compliance policies need a way to run the full `get` -> `generate` -> `scan` pipeline from a local OCI Layout directory without network access or content exposure.

## What Changes

- Introduce `oci-layout://` URL scheme in `complytime.yaml` policy entries, enabling `complyctl get` to sync from a local OCI Layout directory
- Add `LocalSource` implementation of the `PolicySource` interface that reads from a local OCI Layout using `oci.New()` and copies to the cache via `oras.Copy()` (local-to-local, no network)
- Extend `ValidateOCIRef()` to accept `oci-layout://` scheme, bypassing registry host validation while preserving shell metacharacter rejection
- Extend `ParsePolicyRef()` to parse `oci-layout://` URLs into `PolicyRef` with a sentinel `Registry` value for dispatch
- Add scheme dispatch in `syncSinglePolicy()` to route `oci-layout://` URLs to `LocalSource` instead of `RegistrySource`
- No changes to `generate`, `scan`, or the policy loader -- these already operate entirely from the local OCI Layout cache

## Capabilities

### New Capabilities
- `local-oci-source`: Sync policies from local OCI Layout directories via `oci-layout://` scheme in `complyctl get`, enabling private policy evaluation without a registry

### Modified Capabilities

## Impact

- **Code**: `internal/cache/source.go` (new `LocalSource`), `internal/complytime/config.go` (`ValidateOCIRef`, `ParsePolicyRef`), `cmd/complyctl/cli/get.go` (scheme dispatch)
- **Tests**: `internal/cache/source_test.go` (new), `internal/complytime/config_test.go` (extended), E2E test for local layout pipeline
- **Dependencies**: No new dependencies. Uses existing `oras-go/v2/content/oci` for local OCI Layout access (already vendored)
- **No breaking changes**: All existing registry-based behavior is unchanged. The `oci-layout://` scheme is additive.
