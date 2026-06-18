## ADDED Requirements

### Requirement: ParsePolicyRef supports colon-tag syntax
`ParsePolicyRef` SHALL parse OCI references using `:tag` syntax (e.g., `registry.com/org/image:v0.4.0`) and populate `PolicyRef.Version` with the tag value and `PolicyRef.Repository` with the repository path excluding the tag.

#### Scenario: Reference with colon tag
- **WHEN** `ParsePolicyRef` is called with `"quay.io/complytime/complypack-ampel-bp:v0.4.0"`
- **THEN** `PolicyRef.Registry` SHALL be `"quay.io"`, `PolicyRef.Repository` SHALL be `"complytime/complypack-ampel-bp"`, and `PolicyRef.Version` SHALL be `"v0.4.0"`

#### Scenario: Reference with latest tag
- **WHEN** `ParsePolicyRef` is called with `"quay.io/complytime/complypack-ampel-bp:latest"`
- **THEN** `PolicyRef.Registry` SHALL be `"quay.io"`, `PolicyRef.Repository` SHALL be `"complytime/complypack-ampel-bp"`, and `PolicyRef.Version` SHALL be `"latest"`

### Requirement: ParsePolicyRef supports digest syntax
`ParsePolicyRef` SHALL parse OCI references using `@digest` syntax (e.g., `registry.com/org/image@sha256:abc123`) and populate `PolicyRef.Version` with the digest value.

#### Scenario: Reference with SHA256 digest
- **WHEN** `ParsePolicyRef` is called with `"quay.io/complytime/complypack-ampel-bp@sha256:abc123def456"`
- **THEN** `PolicyRef.Registry` SHALL be `"quay.io"`, `PolicyRef.Repository` SHALL be `"complytime/complypack-ampel-bp"`, and `PolicyRef.Version` SHALL be `"sha256:abc123def456"`

### Requirement: ParsePolicyRef supports bare references
`ParsePolicyRef` SHALL parse bare OCI references with no tag or digest and leave `PolicyRef.Version` empty.

#### Scenario: Reference without tag or digest
- **WHEN** `ParsePolicyRef` is called with `"quay.io/complytime/complypack-ampel-bp"`
- **THEN** `PolicyRef.Registry` SHALL be `"quay.io"`, `PolicyRef.Repository` SHALL be `"complytime/complypack-ampel-bp"`, and `PolicyRef.Version` SHALL be `""`

### Requirement: ParsePolicyRef supports bare policy IDs
`ParsePolicyRef` SHALL accept bare policy identifiers with no registry or path separator and populate only `PolicyRef.Repository`.

#### Scenario: Bare policy ID without version
- **WHEN** `ParsePolicyRef` is called with `"nist-800-53-r5"`
- **THEN** `PolicyRef.Registry` SHALL be `""`, `PolicyRef.Repository` SHALL be `"nist-800-53-r5"`, and `PolicyRef.Version` SHALL be `""`

#### Scenario: Bare policy ID with at-version
- **WHEN** `ParsePolicyRef` is called with `"nist-800-53-r5@v1.0"`
- **THEN** `PolicyRef.Registry` SHALL be `""`, `PolicyRef.Repository` SHALL be `"nist-800-53-r5"`, and `PolicyRef.Version` SHALL be `"v1.0"`

### Requirement: ParsePolicyRef supports URL scheme prefixes
`ParsePolicyRef` SHALL strip `http://` and `https://` scheme prefixes and include them in `PolicyRef.Registry`.

#### Scenario: HTTP scheme with port
- **WHEN** `ParsePolicyRef` is called with `"http://localhost:5000/policies/test:v1.0"`
- **THEN** `PolicyRef.Registry` SHALL be `"http://localhost:5000"`, `PolicyRef.Repository` SHALL be `"policies/test"`, and `PolicyRef.Version` SHALL be `"v1.0"`

#### Scenario: HTTPS scheme with tag
- **WHEN** `ParsePolicyRef` is called with `"https://ghcr.io/org/policy:v2.0"`
- **THEN** `PolicyRef.Registry` SHALL be `"https://ghcr.io"`, `PolicyRef.Repository` SHALL be `"org/policy"`, and `PolicyRef.Version` SHALL be `"v2.0"`

### Requirement: ParsePolicyRef returns error for invalid references
`ParsePolicyRef` SHALL return an error for OCI references that are structurally invalid.

#### Scenario: Empty input
- **WHEN** `ParsePolicyRef` is called with `""`
- **THEN** it SHALL return a non-nil error

#### Scenario: Whitespace-only input
- **WHEN** `ParsePolicyRef` is called with `"   "`
- **THEN** it SHALL return a non-nil error

### Requirement: ParsePolicyRef preserves at-version backwards compatibility
`ParsePolicyRef` SHALL continue to support the existing `@version` notation used in policy references.

#### Scenario: Full reference with at-version
- **WHEN** `ParsePolicyRef` is called with `"registry.com/policies/nist-800-53-r5@v1.2.3"`
- **THEN** `PolicyRef.Registry` SHALL be `"registry.com"`, `PolicyRef.Repository` SHALL be `"policies/nist-800-53-r5"`, and `PolicyRef.Version` SHALL be `"v1.2.3"`

### Requirement: Sync functions construct valid lookup references for digests
`SyncPolicy` and `SyncComplypack` SHALL use `@` as the separator when constructing lookup references for digest versions and `:` for tag versions.

#### Scenario: Lookup reference with tag version
- **WHEN** `SyncPolicy` or `SyncComplypack` is called with `repository="org/policy"` and `version="v1.0"`
- **THEN** the lookup reference SHALL be `"org/policy:v1.0"`

#### Scenario: Lookup reference with digest version
- **WHEN** `SyncPolicy` or `SyncComplypack` is called with `repository="org/policy"` and `version="sha256:abc123"`
- **THEN** the lookup reference SHALL be `"org/policy@sha256:abc123"`

### Requirement: Config loading validates OCI references
`LoadFrom` SHALL validate all policy and complypack URLs after loading `complytime.yaml` and return an error if any are structurally invalid.

#### Scenario: Invalid policy URL in config
- **WHEN** `LoadFrom` loads a `complytime.yaml` containing a policy with an invalid OCI reference
- **THEN** it SHALL return an error identifying the invalid entry

#### Scenario: Valid config with mixed reference styles
- **WHEN** `LoadFrom` loads a `complytime.yaml` containing policies using `:tag`, `@version`, and bare references
- **THEN** it SHALL succeed without error
