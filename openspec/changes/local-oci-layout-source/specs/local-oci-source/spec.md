## ADDED Requirements

### Requirement: Accept oci-layout scheme in policy URL
The system SHALL accept `oci-layout://` as a valid URL scheme in `complytime.yaml` policy entries. The URL format SHALL be `oci-layout://<absolute-path>` where `<absolute-path>` is an absolute filesystem path to a directory containing a valid OCI Layout. An optional `@<version>` suffix MAY specify the tag to resolve.

#### Scenario: Valid oci-layout URL passes validation
- **WHEN** `complytime.yaml` contains a policy with `url: oci-layout:///home/user/bundles/my-policy`
- **THEN** `ValidateOCIRef` SHALL accept the URL without error

#### Scenario: oci-layout URL with version suffix
- **WHEN** `complytime.yaml` contains a policy with `url: oci-layout:///home/user/bundles/my-policy@v1.0.0`
- **THEN** `ParsePolicyRef` SHALL extract `Version: "v1.0.0"` and `Repository: "/home/user/bundles/my-policy"`

#### Scenario: Empty path rejected
- **WHEN** `complytime.yaml` contains a policy with `url: oci-layout://`
- **THEN** `ValidateOCIRef` SHALL return an error indicating the path must not be empty

#### Scenario: Shell metacharacters rejected
- **WHEN** `complytime.yaml` contains a policy with `url: oci-layout:///path/with;semicolon`
- **THEN** `ValidateOCIRef` SHALL return an error indicating invalid characters

### Requirement: Sync from local OCI Layout
The system SHALL sync policies from a local OCI Layout directory when the policy URL uses the `oci-layout://` scheme. The sync SHALL use `oras.Copy()` for local-to-local transfer with digest verification. No network access SHALL occur during sync of `oci-layout://` policies.

#### Scenario: Successful sync from local layout
- **WHEN** user runs `complyctl get` with a policy configured as `oci-layout:///path/to/bundle`
- **AND** the path contains a valid OCI Layout with a tagged manifest
- **THEN** the system SHALL copy the manifest and all layers into `~/.complytime/policies/{id}/`
- **AND** the system SHALL update `state.json` with the manifest digest, version, and timestamp

#### Scenario: Incremental sync skips unchanged content
- **WHEN** user runs `complyctl get` with a previously synced `oci-layout://` policy
- **AND** the source OCI Layout manifest digest matches the cached digest in `state.json`
- **THEN** the system SHALL skip the sync and report the policy as up-to-date

#### Scenario: Source layout does not exist
- **WHEN** user runs `complyctl get` with `oci-layout:///nonexistent/path`
- **THEN** the system SHALL return an error indicating the local OCI Layout could not be opened

#### Scenario: Source layout is not a valid OCI Layout
- **WHEN** user runs `complyctl get` with `oci-layout:///path/to/empty-dir`
- **AND** the directory does not contain an `oci-layout` marker file
- **THEN** the system SHALL return an error indicating the path is not a valid OCI Layout

### Requirement: Downstream commands work with locally synced policies
The `generate` and `scan` commands SHALL work without modification for policies synced from `oci-layout://` sources. The commands SHALL read from the local cache at `~/.complytime/policies/{id}/` regardless of the original source scheme.

#### Scenario: Generate after local sync
- **WHEN** user runs `complyctl get` with an `oci-layout://` policy
- **AND** then runs `complyctl generate --policy-id <id>`
- **THEN** the system SHALL resolve the policy from the local cache and generate output

#### Scenario: Mixed sources in single config
- **WHEN** `complytime.yaml` contains both `oci-layout://` and registry-based policy URLs
- **THEN** `complyctl get` SHALL sync each policy from its respective source
- **AND** `complyctl generate` and `complyctl scan` SHALL work for all policies regardless of source

### Requirement: Version resolution for local sources
The system SHALL resolve the manifest tag from the local OCI Layout when no `@version` suffix is specified. The default tag SHALL be `latest`.

#### Scenario: Default version resolution
- **WHEN** the policy URL is `oci-layout:///path/to/bundle` (no version suffix)
- **THEN** `LocalSource.DefinitionVersion` SHALL resolve the `latest` tag from the source layout's `index.json`

#### Scenario: Explicit version resolution
- **WHEN** the policy URL is `oci-layout:///path/to/bundle@v2.0.0`
- **THEN** `LocalSource.DefinitionVersion` SHALL resolve the `v2.0.0` tag from the source layout
