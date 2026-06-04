## Context

The mock OCI registry (`cmd/mock-oci-registry/main.go`) seeds test
content for integration testing. It currently seeds the ampel complypack
(`complypacks/ampel-bp`) with a dummy payload via `buildDummyTarGz()`:

```go
content: buildDummyTarGz("policy.json",
    []byte(`{"name":"ampel-branch-protection","version":"1.0.0"}`))
```

This dummy JSON lacks the `id` field that `LoadGranularPolicies()`
requires. The OPA complypack was already migrated to the
`buildTarGzFromFS()` pattern in commit `74fbae8`, where real policy
files are embedded from `testdata/opa-complypack/` via `//go:embed`.

The cross-repo integration test script (`cross_repo_integration_test.sh`)
currently works around the dummy content by manually copying
`block-force-push.json` into the workspace default directory. With the
ampel provider now consuming `ComplypackContentPath`, this workaround
is insufficient — the complypack content takes precedence.

## Goals / Non-Goals

**Goals:**
- Seed the ampel complypack with valid granular policy content that the
  provider accepts
- Follow the established OPA complypack pattern (embed + `buildTarGzFromFS`)
- Remove the manual granular policy pre-staging from the integration test
- Maintain backward compatibility with the pre-PR#52 ampel provider

**Non-Goals:**
- Adding new ampel policies or expanding test coverage beyond what exists
- Modifying the ampel provider code (that is `complytime-providers` PR #52)
- Changing the complypack OCI artifact format or media types
- Adding OPA-side changes (already handled by `opa-devcontainer-content`)

## Decisions

### D1: Reuse existing `block-force-push.json` content

**Decision**: Copy the content from
`tests/cross-repo/testdata/granular-policies/block-force-push.json`
into `cmd/mock-oci-registry/testdata/ampel-complypack/block-force-push.json`.

**Rationale**: This is the same fixture the integration test already
validates against. Using identical content ensures the test assertions
(e.g., checking for `block-force-push` policy ID in results) continue
to pass without modification.

**Alternatives considered**:
- Create minimal stub content: Rejected — would require updating test
  assertions and diverges from the real policy format the test validates.

### D2: Remove pre-staged granular policies from integration test

**Decision**: Remove the `mkdir -p` and `cp` lines that pre-stage
`block-force-push.json` into `.complytime/ampel/granular-policies/` in
`cross_repo_integration_test.sh`.

**Rationale**: With `ComplypackContentPath` taking precedence in PR #52's
provider, the pre-staged directory is never consulted. Keeping it creates
a false safety net — the test would pass even if complypack delivery
broke, defeating the purpose of the integration test.

**Alternatives considered**:
- Keep pre-staged content as fallback: Rejected — masks complypack
  delivery failures and makes the test less meaningful.

### D3: Add `//go:embed` directive alongside the existing OPA one

**Decision**: Add a new `//go:embed testdata/ampel-complypack/*` var
declaration directly below the existing OPA embed directive.

**Rationale**: Follows the established pattern. Each complypack provider
gets its own embedded filesystem variable and testdata subdirectory.

## Risks / Trade-offs

- **[Risk] Merge ordering**: If this PR merges before
  `complytime-providers` PR #52, the cross-repo CI will build the old
  ampel provider from `main`, which ignores `ComplypackContentPath`
  and falls back to the default directory. Since we're removing the
  pre-staged content, the test would fail.
  → **Mitigation**: Keep the pre-staged content removal as a separate,
  final task. If needed, split the change: merge the complypack content
  seeding first (backward-compatible), then remove pre-staging after
  PR #52 lands on `complytime-providers` main.

- **[Risk] Content drift**: The embedded `block-force-push.json` could
  diverge from the provider's expected format over time.
  → **Mitigation**: This is test content for integration validation —
  if the format changes, the integration test will catch it (by design).
