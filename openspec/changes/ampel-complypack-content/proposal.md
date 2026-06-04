## Why

The mock OCI registry seeds the `complypacks/ampel-bp` artifact with a
dummy `policy.json` containing `{"name":"...","version":"..."}` — a
placeholder that lacks the `id` field required by the ampel provider's
`LoadGranularPolicies()`. Before `complytime-providers` PR #52, the ampel
provider ignored `ComplypackContentPath` and fell back to pre-staged
granular policies, so the dummy content was never parsed. Now that the
provider consumes complypack content, the cross-repo integration test
fails because the dummy payload is rejected. This change replaces the
dummy content with a valid ampel granular policy, mirroring the pattern
already established for the OPA complypack (commit `74fbae8`).

## What Changes

- Replace the dummy `buildDummyTarGz("policy.json", ...)` call in
  `seedDefaults()` with `buildTarGzFromFS()` using embedded ampel
  complypack test content
- Add `testdata/ampel-complypack/block-force-push.json` containing a
  valid `AmpelPolicy` structure (matching the existing cross-repo test
  fixture at `tests/cross-repo/testdata/granular-policies/`)
- Add `//go:embed testdata/ampel-complypack/*` directive to embed the
  ampel complypack content
- Update `TestSeedDefaults_AllReposSeeded` to verify the ampel
  complypack contains the expected file count and content structure
- Remove the now-unused `block-force-push.json` from
  `tests/cross-repo/testdata/granular-policies/` and the corresponding
  `cp` / `mkdir` lines from `cross_repo_integration_test.sh`, since
  the provider will consume complypack content directly instead of the
  pre-staged fallback directory

## Capabilities

### New Capabilities
- `ampel-complypack-seed`: Embedded ampel complypack test content in the
  mock OCI registry, producing a valid tar.gz payload that the ampel
  provider's `LoadGranularPolicies()` accepts

### Modified Capabilities

## Impact

- `cmd/mock-oci-registry/main.go`: New embed directive, updated
  `seedDefaults()` call for `complypacks/ampel-bp`
- `cmd/mock-oci-registry/testdata/ampel-complypack/`: New directory with
  valid granular policy JSON
- `cmd/mock-oci-registry/main_test.go`: Updated test assertions for
  ampel complypack content
- `tests/cross-repo/testdata/granular-policies/`: Removed (content
  migrated to embedded complypack)
- `tests/cross-repo/cross_repo_integration_test.sh`: Simplified setup
  (no manual granular policy staging)
- Backward-compatible: the old ampel provider ignores
  `ComplypackContentPath` and is unaffected by serving valid content
