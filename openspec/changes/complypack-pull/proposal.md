## Why

Users need to pull published complypacks from OCI registries into their local environment so that complyctl provider plugins can consume them. Today, complyctl fetches Gemara policies (control catalogs, guidelines, assessment plans) via `complyctl get`, but there is no mechanism to fetch the evaluator-specific assessment logic that providers need to actually execute scans. Complypacks fill this gap -- they package opaque policy content (e.g., OPA bundles, CEL rulesets) alongside an `evaluator-id` that identifies which provider plugin should consume them.

The critical design constraint is **content-type driven dispatch**: the CLI MUST dispatch to the correct validator, tester, and generator based on the `evaluator-id` in the complypack config, not by file extension or hardcoded assumption. This ensures the system remains evaluator-agnostic and extensible to new policy languages without code changes.

The `github.com/complytime/complypack` Go library provides the Pack/Unpack primitives and OCI media type definitions. This change integrates that library into complyctl to pull, cache, and dispatch complypacks to providers.

## What Changes

- **Add `complypack` dependency**: Import `github.com/complytime/complypack/pkg/complypack` into complyctl for Unpack, Config, and media type constants.
- **Add `complyctl get` complypack support**: Extend the existing `get` command to recognize complypack artifacts in OCI registries (by `artifactType: application/vnd.complypack.artifact.v1`) and pull them into a local cache alongside existing Gemara policy artifacts.
- **Complypack cache**: Store unpacked complypacks under `~/.complytime/complypacks/{evaluator-id}/{version}/` with the opaque content and a metadata file containing the Config (evaluator-id, version, provenance).
- **Evaluator-ID dispatch**: When the scan pipeline invokes Generate, route complypack content to the provider whose `evaluator-id` matches the complypack's `evaluator-id`. The complypack lookup happens in the CLI layer (alongside the existing `policy.GroupByEvaluator` pre-processing), with `Manager.GetProvider(evaluatorID)` handling the actual provider dispatch.
- **Provider protocol extension**: Extend the `GenerateRequest` protobuf message to include an optional complypack content path, so providers can locate their assessment logic without hardcoding paths.
- **`complyctl providers` enhancement**: Show which complypacks are available for each discovered provider, based on evaluator-id matching.
- **`complyctl doctor` enhancement**: Add a diagnostic check that verifies each configured policy's evaluator-id has a matching complypack cached locally.
- **Workspace config extension**: Add an optional `complypacks` section to `complytime.yaml` for declaring complypack OCI references, parallel to the existing `policies` section.

## Capabilities

### New Capabilities
- `complypack-pull`: Pull published complypack OCI artifacts from registries into the local cache, dispatched by evaluator-id.
- `complypack-cache`: Local filesystem cache for unpacked complypack content, organized by evaluator-id and version.
- `complypack-dispatch`: Content-type driven routing of complypack content to the correct provider plugin based on evaluator-id.

### Modified Capabilities
- `get`: Extended to pull complypack artifacts in addition to Gemara policy artifacts. Detection is by OCI `artifactType`, not file extension.
- `scan`: Generate phase extended to pass complypack content path to providers via the `GenerateRequest`.
- `providers`: Enhanced to show matched complypack availability per provider.
- `doctor`: Enhanced with complypack availability diagnostics.
- `complytime.yaml`: Extended with optional `complypacks` section.
- `plugin.proto`: `GenerateRequest` extended with optional complypack content path field.

### Removed Capabilities
- None. This is purely additive.

## Impact

- **CLI surface**: `complyctl get` gains complypack awareness. No new top-level commands; complypack pull is integrated into the existing `get` workflow.
- **Code**: `cmd/complyctl/cli/get.go`, `cmd/complyctl/cli/scan.go`, `cmd/complyctl/cli/providers.go`, `cmd/complyctl/cli/doctor.go`, `internal/complytime/config.go`, `internal/cache/`, `pkg/provider/client.go`, `pkg/provider/server.go`, `api/plugin/plugin.proto`.
- **Dependencies**: New Go module dependency on `github.com/complytime/complypack`.
- **Tests**: Unit tests for complypack cache operations, evaluator-id dispatch logic, config parsing. E2E tests for pull-and-scan workflow with test-provider. Integration tests for OCI artifact type detection.
- **Documentation**: `docs/QUICK_START.md` workflow steps, `docs/man/complyctl.md` commands section, `complytime.yaml` reference, command help text updates.
- **Wire protocol**: Additive protobuf field (backward compatible). Existing providers that ignore the new field continue to work unchanged.

## Constitution Alignment

### I. Single Source of Truth (Centralized Constants)

**Assessment**: PASS

Complypack media types come from the `complypack` library's `MediaTypeArtifact`, `MediaTypeConfig`, and `MediaTypeContent` constants -- no magic strings in complyctl. The evaluator-id is the single dispatch key, read from the complypack config, not inferred from filenames or paths.

### II. Simplicity & Isolation

**Assessment**: PASS

The complypack cache is a separate package from the existing policy cache. Dispatch logic extends the existing `GroupByEvaluator` pattern rather than introducing a new routing mechanism. Each concern (pull, cache, dispatch) is isolated into focused functions.

### III. Incremental Improvement

**Assessment**: PASS

This change adds complypack support without refactoring existing Gemara policy handling. The `get` command gains a new code path for complypack artifacts but the existing policy sync flow is untouched. No unrelated changes are bundled.

### IV. Readability First

**Assessment**: PASS

The evaluator-id dispatch pattern is already established in the codebase (`GroupByEvaluator`, `Manager.RouteScan`). Extending it to complypacks uses the same vocabulary and routing model that contributors already understand.

### V. Do Not Reinvent the Wheel

**Assessment**: PASS

Uses the `github.com/complytime/complypack` library for Pack/Unpack operations rather than implementing OCI artifact handling from scratch. The library is maintained by the complytime org, uses ORAS (the same OCI library complyctl already depends on), and has CI/linting/tests.

### VI. Composability (The Unix Philosophy)

**Assessment**: PASS

Complypacks are opaque content containers -- the complyctl host does not interpret the content, it routes it to the correct provider based on evaluator-id. This preserves the plugin boundary: new evaluator types (CEL, CUE, etc.) require only a new provider binary, not complyctl changes.

### VII. Convention Over Configuration

**Assessment**: PASS

Cache location follows the existing convention (`~/.complytime/complypacks/`). Complypack artifacts are auto-detected by OCI `artifactType` during `get`, requiring no additional user configuration when complypacks are published alongside policies in the same registry.
