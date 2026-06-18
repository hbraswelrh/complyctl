## Context

`ParsePolicyRef` is a hand-rolled OCI reference parser in `internal/complytime/config.go` used by 10 production call sites across `cmd/` and `internal/`. It only recognizes `@` as a version separator, missing the standard `:tag` syntax. The codebase already vendors `oras.land/oras-go/v2` (v2.6.1) which includes `registry.ParseReference` — a robust parser that handles all four OCI reference forms (`:tag`, `@digest`, `:tag@digest`, and bare).

The downstream sync functions (`SyncPolicy`, `SyncComplypack`) construct `lookupRef` by concatenating `repository + ":" + version`, which produces invalid double-tagged references when the tag was not stripped from the repository during parsing.

## Goals / Non-Goals

**Goals:**
- Support `:tag`, `@digest`, and bare OCI references in both `policies` and `complypacks` config entries
- Return validation errors for malformed OCI references
- Fail fast at config load time with clear error messages
- Maintain backwards compatibility with existing `@version` notation and bare policy IDs

**Non-Goals:**
- Distinguishing digest from tag at the `PolicyRef` struct level (keep single `Version` field)
- Changing `EffectiveID()` signature
- Refactoring the sync functions beyond the `lookupRef` construction fix
- Supporting `registry/repo:tag@digest` form (oras-go silently drops the tag in this case, which is acceptable)

## Decisions

### 1. Delegate to oras-go `registry.ParseReference` for OCI refs

**Decision:** Strip any URL scheme prefix, then delegate to `registry.ParseReference` for inputs containing a `/`. For bare IDs (no `/`), handle directly as repository-only refs.

**Rationale:** oras-go's parser is battle-tested, handles all four OCI forms correctly, and is already vendored. Reimplementing this logic is the source of the current bug.

**Alternatives considered:**
- Fix the hand-rolled parser to also handle `:tag` — viable but fragile, duplicates logic that oras-go already provides.
- Replace `PolicyRef` with oras-go's `Reference` directly — too invasive, loses the scheme handling and bare-ID support that `ParsePolicyRef` provides.

### 2. Keep `Version` field as-is (Option 3 from exploration)

**Decision:** Map oras-go's `Reference.Reference` field directly to `PolicyRef.Version`. Callers that construct OCI references check `strings.HasPrefix(version, "sha256:")` to decide between `@` and `:` separators.

**Rationale:** Minimal struct change, no new fields. The digest-vs-tag distinction only matters in two places (the sync `lookupRef` construction), and a prefix check is clear and sufficient.

### 3. Validate at `LoadFrom` time

**Decision:** After unmarshaling `complytime.yaml`, iterate all `Policies` and `Complypacks` entries and call `ParsePolicyRef` on each URL. Return an error if any are invalid.

**Rationale:** Fail fast with a clear error message. Downstream code (including `EffectiveID()`) can then assume valid refs without propagating errors. This avoids changing `EffectiveID()` to return `(string, error)`, which would cascade through many callers.

### 4. Extract `lookupRef` construction into a helper

**Decision:** Add a small helper (e.g., `buildLookupRef(repository, version string) string`) that encapsulates the tag-vs-digest logic, used by both `SyncPolicy` and `SyncComplypack`.

**Rationale:** Both sync functions have identical `lookupRef` construction logic. Centralizing it prevents the digest handling from being duplicated.

## Risks / Trade-offs

- **[Bare IDs bypass oras-go validation]** → Bare IDs like `nist-800-53-r5` are intentionally passed through without oras-go validation. These are convention-based identifiers, not OCI refs. If stricter validation is needed later, it can be added separately.
- **[Scheme stripping before oras-go]** → `ParsePolicyRef` strips `http://`/`https://` before passing to oras-go, then reattaches the scheme to `Registry`. If oras-go ever changes how it validates the registry component, this could interact poorly. Mitigation: the scheme handling is straightforward string manipulation with existing test coverage.
- **[`EffectiveID()` trusts pre-validation]** → If `ParsePolicyRef` is called on an unvalidated string outside of config loading, `EffectiveID()` could silently produce wrong IDs. Mitigation: `ParsePolicyRef` returns an error, so any new call site will be prompted by the compiler to handle it.
