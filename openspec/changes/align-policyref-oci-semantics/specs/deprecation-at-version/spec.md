## ADDED Requirements

### Requirement: Deprecation warning for @version notation
`ParsePolicyRef` SHALL emit a deprecation warning to stderr when
a full OCI reference (containing at least one `/`) uses a
non-digest `@` suffix. The warning SHALL identify the offending
URL, advise switching to `:tag` syntax, and note that `@version`
support will be removed in a future release.

Bare policy IDs (no `/` in the input) are exempt from this warning
because the `@` separator is a complytime convention unrelated to
OCI digest semantics.

#### Scenario: Full OCI reference with @version emits warning
- **WHEN** `ParsePolicyRef` is called with
  `"registry.example.com/policies/nist@v1.0"`
- **THEN** a deprecation warning is written to stderr containing
  the original URL and recommending `:tag` syntax
- **AND** the function returns a valid `PolicyRef` with
  `Tag == "v1.0"` (backwards-compatible behavior preserved)

#### Scenario: Full OCI reference with @digest does not warn
- **WHEN** `ParsePolicyRef` is called with
  `"registry.example.com/policies/nist@sha256:abc123..."`
- **THEN** no deprecation warning is emitted
- **AND** the function returns a valid `PolicyRef` with the
  `Digest` field populated

#### Scenario: Bare policy ID with @version does not warn
- **WHEN** `ParsePolicyRef` is called with `"my-policy@v2.0"`
- **THEN** no deprecation warning is emitted
- **AND** the function returns a valid `PolicyRef` with
  `Tag == "v2.0"`

### Requirement: CheckPolicyActivePeriod reports parse failures
`CheckPolicyActivePeriod` SHALL report `ParsePolicyRef` failures
as a `CheckResult` with `Status: StatusFail` and
`Blocking: true`, matching the error-reporting pattern in
`CheckPolicyVersions`.

#### Scenario: Malformed policy URL in active period check
- **WHEN** `CheckPolicyActivePeriod` processes a policy whose
  URL causes `ParsePolicyRef` to return an error
- **THEN** the results include a `CheckResult` with
  `Status == StatusFail`, `Blocking == true`, and a message
  containing the parse error

### Requirement: CheckComplypacks reports parse failures
`CheckComplypacks` SHALL report `ParsePolicyRef` failures as a
`CheckResult` with `Status: StatusFail` and `Blocking: true`,
matching the error-reporting pattern in `CheckPolicyVersions`.

#### Scenario: Malformed policy URL in complypacks check
- **WHEN** `CheckComplypacks` processes a policy whose URL
  causes `ParsePolicyRef` to return an error
- **THEN** the results include a `CheckResult` with
  `Status == StatusFail`, `Blocking == true`, and a message
  containing the parse error

### Requirement: CheckVariables reports resolution failures
`CheckVariables` SHALL report specific error messages for each
policy resolution failure instead of silently incrementing a
counter. Each failure SHALL produce a `CheckResult` with
`Status: StatusWarn` and `Blocking: false`.

#### Scenario: Policy not found in config during variable check
- **WHEN** `CheckVariables` processes a target referencing a
  policy ID that does not exist in `cfg.Policies`
- **THEN** the results include a `CheckResult` with
  `Status == StatusWarn` and a message identifying the missing
  policy ID

#### Scenario: Parse error during variable check
- **WHEN** `CheckVariables` processes a policy whose URL causes
  `ParsePolicyRef` to return an error
- **THEN** the results include a `CheckResult` with
  `Status == StatusWarn` and a message containing the parse error

#### Scenario: Version resolution failure during variable check
- **WHEN** `CheckVariables` processes a policy whose version
  cannot be resolved by the resolver
- **THEN** the results include a `CheckResult` with
  `Status == StatusWarn` and a message identifying the policy
  and the resolution error

#### Scenario: Graph resolution failure during variable check
- **WHEN** `CheckVariables` processes a policy whose dependency
  graph cannot be resolved
- **THEN** the results include a `CheckResult` with
  `Status == StatusWarn` and a message identifying the policy
  and the resolution error
