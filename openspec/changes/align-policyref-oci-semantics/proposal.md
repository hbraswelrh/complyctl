## Why

PR #595 improved OCI reference parsing by delegating to oras-go and
PR branch `opsx/fix-complypack-oci-ref` split `PolicyRef.Version`
into typed `Tag` and `Digest` fields. Two follow-up improvements
remain to complete alignment with OCI Distribution Spec semantics
(tracked in issue #600):

1. The `@version` notation (e.g., `registry.com/repo@v1.0`) silently
   converts to `:tag` syntax without warning users. In OCI semantics,
   `@` is exclusively a digest separator. Users familiar with OCI
   tooling may find this ambiguous.
2. Three `complyctl doctor` check functions silently skip policies
   that fail to parse or resolve, producing no diagnostic output.
   This hides configuration errors from users.

## What Changes

- Emit a deprecation warning to stderr when `ParsePolicyRef`
  encounters a non-digest `@` suffix on a full OCI reference
  (e.g., `@v1.0`), advising the user to switch to `:tag` syntax.
- Update `CheckPolicyActivePeriod` to report `ParsePolicyRef`
  failures as `CheckResult{Status: StatusFail}`, matching the
  error-reporting pattern established in `CheckPolicyVersions`.
- Update `CheckComplypacks` to report `ParsePolicyRef` failures
  as `CheckResult{Status: StatusFail}`, matching the same pattern.
- Update `CheckVariables` to include specific error messages when
  policy resolution fails, instead of silently incrementing a
  counter.

## Capabilities

### New Capabilities

- `deprecation-at-version`: Deprecation warning for the non-standard
  `@version` notation on OCI references, guiding users toward
  standard `:tag` syntax.

### Modified Capabilities

- `oci-ref-parsing`: `ParsePolicyRef` gains a deprecation warning
  side-effect for `@version` notation on full OCI references.

## Impact

- `internal/complytime/config.go` — `ParsePolicyRef` emits
  deprecation warning to stderr for non-digest `@` suffixes
- `internal/complytime/config_test.go` — tests for deprecation
  warning output
- `internal/doctor/doctor.go` — `CheckPolicyActivePeriod`,
  `CheckComplypacks`, and `CheckVariables` gain explicit error
  reporting for parse and resolve failures
- `internal/doctor/doctor_test.go` — tests for new error-reporting
  paths
- `docs/` — document the `@version` deprecation
- `CHANGELOG.md` — deprecation entry
