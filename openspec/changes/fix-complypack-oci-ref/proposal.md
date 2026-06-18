## Why

`ParsePolicyRef` in `internal/complytime/config.go` only recognizes `@` as a version separator, not the standard OCI `:tag` syntax. When a complypack (or policy) URL uses `:v0.4.0`, the tag is left embedded in the `Repository` field. Downstream sync functions then append `":" + version`, producing a double-tagged reference like `org/pack:v0.4.0:v0.4.0` which fails as an invalid OCI reference. The `:latest` tag works only by accident due to a guard that skips appending when the version is `"latest"`. This is tracked in [issue #594](https://github.com/complytime/complyctl/issues/594).

## What Changes

- Rewrite `ParsePolicyRef` to delegate OCI reference parsing to the vendored `oras.land/oras-go/v2/registry.ParseReference`, which correctly handles `:tag`, `@digest`, and bare references.
- Change `ParsePolicyRef` signature from `ParsePolicyRef(raw string) PolicyRef` to `ParsePolicyRef(raw string) (PolicyRef, error)`, returning validation errors for malformed OCI references.
- Add digest-aware reference construction in `SyncPolicy` and `SyncComplypack` so digest versions use `@` instead of `:` when building `lookupRef`.
- Add config-time validation in `LoadFrom` to fail fast on invalid policy/complypack URLs with clear error messages.
- Update all 10 production callers of `ParsePolicyRef` to handle the new error return.

## Capabilities

### New Capabilities
- `oci-ref-parsing`: Standard OCI reference parsing for policy and complypack URLs, supporting `:tag`, `@digest`, and bare repository forms with validation.

### Modified Capabilities

## Impact

- `internal/complytime/config.go` — `ParsePolicyRef` signature and implementation change; `LoadFrom` gains validation loop
- `internal/complytime/config_test.go` — existing tests updated for error return; new test cases for `:tag` and `@digest`
- `internal/cache/sync.go` — `lookupRef` construction handles digest references
- `internal/cache/complypack_sync.go` — `lookupRef` construction handles digest references
- `cmd/complyctl/cli/get.go` — 2 call sites handle `ParsePolicyRef` error
- `cmd/complyctl/cli/scan.go` — 1 call site handles `ParsePolicyRef` error
- `cmd/complyctl/cli/list.go` — 1 call site handles `ParsePolicyRef` error
- `cmd/complyctl/cli/generate.go` — 1 call site handles `ParsePolicyRef` error
- `internal/doctor/doctor.go` — 4 call sites handle `ParsePolicyRef` error
- `EffectiveID()` is unchanged — config validation at load time ensures it never receives invalid refs
