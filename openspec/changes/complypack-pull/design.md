## Context

Complyctl currently fetches Gemara policy artifacts (control catalogs, guidelines, assessment plans) from OCI registries via `complyctl get` and caches them as OCI Layouts under `~/.complytime/policies/`. Providers consume these policies during the Generate/Scan pipeline, but there is no mechanism for fetching the evaluator-specific assessment logic (e.g., OPA bundles, CEL rulesets) that providers need to execute scans.

The `github.com/complytime/complypack` library provides a standardized OCI artifact format for this assessment logic. A complypack is an OCI Image Manifest v1.1 artifact with:
- `artifactType: application/vnd.complypack.artifact.v1`
- A config layer (`application/vnd.complypack.config.v1+json`) containing `evaluator-id`, `version`, and optional provenance
- A content layer (`application/vnd.complypack.content.v1.tar+gzip`) containing opaque evaluator-specific content

The `evaluator-id` in the config is the dispatch key -- it determines which provider plugin consumes the content. This is the same evaluator-id concept already used throughout complyctl for routing Generate/Scan requests.

## Goals / Non-Goals

**Goals:**
- Pull complypack OCI artifacts from registries into a local cache
- Dispatch complypack content to providers based on `evaluator-id` from the complypack config (content-type driven dispatch)
- Extend `GenerateRequest` with an optional complypack content path for providers
- Reuse existing OCI infrastructure (oras-go, Docker credential chain, incremental sync)
- Maintain backward compatibility -- providers that don't use complypacks continue to work unchanged

**Non-Goals:**
- Building or publishing complypacks (handled by the `complypack` CLI)
- Signing or verifying complypack signatures (deferred -- the complypack library supports it but the feature is not yet implemented)
- Adding a new top-level CLI command (complypack pull integrates into existing `get`)
- Modifying provider-side logic (providers choose whether to use the complypack path)
- MCP server integration (out of scope for this change)

## Decisions

### D1: Integrate into `complyctl get` rather than a new command

Complypack pull is handled by the existing `complyctl get` command rather than a new `complyctl pack pull` or similar. The `get` command already manages OCI artifact fetching, credential resolution, and incremental sync. Adding complypack fetching as a parallel loop in the same command keeps the user workflow simple: one command fetches everything.

Rationale: Convention over Configuration (Principle VII). Users already run `complyctl get` to prepare their environment. Adding a second fetch command creates unnecessary ceremony.

### D2: Complypack cache at `~/.complytime/complypacks/{evaluator-id}/{version}/`

Complypacks are cached separately from Gemara policies because they serve a different purpose and have a different lifecycle. The evaluator-id is the top-level directory key because it is the dispatch key -- when the scan pipeline needs to find complypack content for a provider, it looks up by evaluator-id. Version is the second-level key to support multiple versions.

Each version directory contains:
- `content.tar.gz` -- the opaque content blob from the OCI content layer (stored as-is, not extracted by complyctl)
- `config.json` -- the complypack Config (evaluator-id, version, provenance) for metadata access without re-parsing the OCI manifest

The `evaluator-id` and `version` values from the complypack config are validated before use as path components. Values must match `[a-zA-Z0-9._-]+` (alphanumeric, dots, hyphens, underscores). Values containing path separators, `..`, or null bytes are rejected. This prevents path traversal attacks from malicious complypack configs.

Cache writes are atomic: content and metadata are written to a temporary directory first, then renamed to the final location. State is only updated after successful rename. If the process is interrupted, the cache remains in a consistent state.

Alternative considered: Storing complypacks as OCI Layouts (same as policies). Rejected because complypacks need to be unpacked for providers to access the content directly via filesystem path, and the evaluator-id keying makes dispatch O(1) without scanning manifests.

### D3: `complypacks` section in `complytime.yaml`

A new optional `complypacks` section is added to `WorkspaceConfig`, structurally identical to `policies`:

```yaml
complypacks:
  - url: ghcr.io/complytime/complypack-opa@v1.0.0
    id: opa-bundle  # optional shortname
```

This reuses the existing `PolicyEntry` type and validation logic (`ValidateOCIRef`, uniqueness checks, `EffectiveID` derivation). The section is optional -- workspaces without complypacks continue to work. Implementation note: consider renaming `PolicyEntry` to a more generic `OCIEntry` or `ArtifactEntry` (with `PolicyEntry` as a type alias for backward compatibility) to avoid semantic confusion when the same type is used for non-policy artifacts.

Rationale: Single Source of Truth (Principle I) -- the same data type and validation for both OCI reference lists. Composability (Principle VI) -- complypacks are independent of policies.

### D4: Complypack content path passed via `GenerateRequest`

The `GenerateRequest` protobuf message gets a new optional `string complypack_content_path` field. The provider manager resolves the complypack for the target evaluator-id, determines the filesystem path, and populates the field. Providers that need complypack content read from this path. Providers that don't need it ignore the empty string.

This is an additive protobuf change -- new fields with default zero values are wire-compatible with existing providers.

Alternative considered: Environment variable (`COMPLYPACK_PATH`). Rejected because it would require per-evaluator env vars or a single shared path, and the provider manager already owns the dispatch decision.

Alternative considered: Streaming content over gRPC. Rejected because providers already have filesystem access to `~/.complytime/` and streaming adds complexity for large bundles.

### D5: Use `complypack.Unpack` for content extraction

The complypack library's `Unpack` function handles manifest parsing, media type validation, and content extraction. Complyctl calls `Unpack` after fetching the artifact into an OCI Layout store, then writes `ComplyPack.Content` to the cache directory and serializes `ComplyPack.Config` to `config.json`.

Rationale: Do Not Reinvent the Wheel (Principle V) -- the library handles media type validation and config deserialization.

### D6: Detection by `artifactType`, not file extension

During `complyctl get`, complypack artifacts are distinguished from Gemara policy artifacts by the OCI manifest's `artifactType` field. Gemara policies use `application/vnd.gemara.*` media types on their config layer. Complypacks use `application/vnd.complypack.artifact.v1` as the manifest `artifactType`. This is inspected after fetching the manifest, not by URL pattern or file extension.

Rationale: This is the core content-type driven dispatch requirement. The `artifactType` is the OCI-standard mechanism for artifact classification.

### D7: State tracking extends existing `cache.State` struct

Complypack sync state (version, digest, timestamp) is tracked by extending the existing `State` struct in `cache/state.go` with a `Complypacks map[string]PolicyState` field (or rename `PolicyState` to `ArtifactState` to reflect shared use). State remains in the single `~/.complytime/state.json` file. This avoids creating a parallel state tracking mechanism and keeps state management centralized. The existing `LoadState`/`SaveState` functions remain the single entry point. This enables incremental sync -- if the remote manifest digest matches the cached digest, the complypack is not re-fetched.

### D8: Protobuf field number for `complypack_content_path`

The new `complypack_content_path` field is added as the next available field number in `GenerateRequest` in `api/plugin/plugin.proto`. After adding the field, `plugin.pb.go` and `plugin_grpc.pb.go` are regenerated.

## Risks / Trade-offs

- **[New dependency]** Adding `github.com/complytime/complypack` as a Go module dependency. The library depends on oras-go and go-gemara, both of which complyctl already vendors. Risk is low -- the dependency tree is compatible.
- **[Cache disk usage]** Complypack content (up to 100MB per artifact) is stored unpacked on disk. No automatic cleanup is implemented in this change. Acceptable for initial implementation; cache eviction can be added later.
- **[Provider adoption]** Existing providers will receive a `complypack_content_path` field they don't use. This is benign -- empty string default, no behavior change. Providers opt in by reading the field.
- **[No signature verification]** The complypack library's signing/verification is not yet implemented (returns "not yet implemented" errors). This change does not attempt to verify complypack signatures. A WARNING is logged at pull time to alert users that the artifact has not been cryptographically verified. When the library adds support, complyctl can adopt it.
- **[No retry logic]** Transient network errors during large complypack pulls will fail the `complyctl get` operation without retry. This is consistent with the existing policy sync behavior -- retry-with-backoff is a codebase-wide improvement deferred to a future change.
- **[Content not extracted by complyctl]** The `content.tar.gz` blob is stored as-is in the cache and passed to providers via `complypack_content_path`. Complyctl does not extract the tar archive -- extraction safety (zip-slip protection) is the provider's responsibility.
