## ADDED Requirements

### Requirement: Workspace config declares complypack references
The `complytime.yaml` workspace configuration SHALL support an optional `complypacks` section containing a list of OCI references to complypack artifacts. Each entry SHALL have a `url` field (OCI reference) and an optional `id` field (user-chosen shortname). The `complypacks` section SHALL follow the same structure and validation rules as the existing `policies` section (OCI ref format, uniqueness constraints, env var expansion).

#### Scenario: Config with complypacks section
- **GIVEN** a `complytime.yaml` with a `complypacks` section containing one or more OCI references
- **WHEN** the config is loaded and validated
- **THEN** the complypack entries SHALL be parsed into the same `PolicyEntry` structure used for policies, with effective IDs derived from the URL when `id` is omitted

#### Scenario: Config without complypacks section
- **GIVEN** a `complytime.yaml` without a `complypacks` section
- **WHEN** the config is loaded and validated
- **THEN** validation SHALL pass and the complypacks list SHALL be empty

#### Scenario: Duplicate complypack URL
- **GIVEN** a `complytime.yaml` with two complypack entries sharing the same `url`
- **WHEN** the config is validated
- **THEN** validation SHALL fail with a duplicate URL error

### Requirement: Get command pulls complypack artifacts
The `complyctl get` command SHALL pull complypack OCI artifacts declared in the `complypacks` section of `complytime.yaml` from OCI registries into a local cache. Detection of complypack artifacts SHALL be by the OCI `artifactType` field (`application/vnd.complypack.artifact.v1`), not by file extension or hardcoded assumption.

#### Scenario: Pull complypack from registry
- **GIVEN** a `complytime.yaml` with a complypack entry pointing to a valid OCI registry
- **WHEN** the user runs `complyctl get`
- **THEN** the complypack artifact SHALL be fetched from the registry, unpacked using the `complypack` library's `Unpack` function, and its content stored in the local cache

#### Scenario: Pull complypack with authentication
- **GIVEN** a complypack hosted on a registry requiring authentication
- **WHEN** the user runs `complyctl get` with valid Docker credentials configured
- **THEN** the complypack SHALL be fetched using the same Docker credential chain as policy fetches

#### Scenario: Incremental sync
- **GIVEN** a complypack already cached locally with an up-to-date digest
- **WHEN** the user runs `complyctl get`
- **THEN** the complypack SHALL NOT be re-fetched; the existing cache SHALL be reused

#### Scenario: Unsigned artifact warning
- **GIVEN** a complypack artifact without cryptographic signature verification (library verification not yet implemented)
- **WHEN** the user runs `complyctl get` and the complypack is fetched
- **THEN** the command SHALL log a WARNING indicating the complypack artifact has not been cryptographically verified

#### Scenario: Atomic cache write
- **GIVEN** a complypack fetch is in progress
- **WHEN** the process is interrupted mid-write (e.g., Ctrl-C, power failure)
- **THEN** the cache SHALL remain in a consistent state; incomplete cache entries SHALL NOT be treated as up-to-date on subsequent runs. Cache writes SHALL use a temporary directory and atomic rename to the final path. State SHALL only be updated after successful cache write.

### Requirement: Complypack cache organized by evaluator-id
Unpacked complypack content SHALL be stored under `~/.complytime/complypacks/{evaluator-id}/{version}/` with the opaque content and a metadata JSON file containing the complypack `Config` (evaluator-id, version, provenance). The evaluator-id in the cache path SHALL be read from the complypack config, not inferred from the OCI reference or filename. The `evaluator-id` and `version` values SHALL be validated against a safe character set before use as filesystem path components. Values containing path separators (`/`, `\`), parent directory references (`..`), or null bytes SHALL be rejected with an error.

#### Scenario: Cache directory structure
- **GIVEN** a complypack with `evaluator-id: io.complytime.opa` and `version: 1.0.0`
- **WHEN** the complypack is pulled and cached
- **THEN** the content SHALL be stored at `~/.complytime/complypacks/io.complytime.opa/1.0.0/content.tar.gz` and metadata at `~/.complytime/complypacks/io.complytime.opa/1.0.0/config.json`

#### Scenario: Multiple evaluator versions
- **GIVEN** two complypacks with the same evaluator-id but different versions
- **WHEN** both are pulled
- **THEN** each SHALL be stored in its own version subdirectory under the evaluator-id directory

#### Scenario: Malicious evaluator-id rejected
- **GIVEN** a complypack with `evaluator-id: ../../etc/malicious` or containing path separators, `..`, or null bytes
- **WHEN** the complypack is unpacked
- **THEN** the command SHALL reject the artifact with a descriptive error and SHALL NOT write any files to the cache

### Requirement: Content-type driven dispatch to providers
The scan pipeline SHALL dispatch complypack content to the provider whose `evaluator-id` matches the complypack's `evaluator-id` from its config. Dispatch MUST be based on the `evaluator-id` field in the complypack config, not by file extension, filename pattern, or hardcoded assumption. When multiple complypacks exist for the same evaluator-id, the version matching the workspace config entry SHALL be used.

#### Scenario: Single evaluator dispatch
- **GIVEN** a cached complypack with `evaluator-id: io.complytime.opa` and a loaded provider with matching evaluator-id
- **WHEN** the scan pipeline invokes Generate
- **THEN** the complypack content path SHALL be passed to the `io.complytime.opa` provider via the `GenerateRequest`

#### Scenario: No matching provider for cached complypack
- **GIVEN** a cached complypack with `evaluator-id: io.complytime.cel` and no loaded provider with that evaluator-id
- **WHEN** the scan pipeline attempts dispatch
- **THEN** the cached complypack SHALL be silently skipped; the scan pipeline SHALL NOT error for orphaned complypacks that have no matching provider

#### Scenario: No complypack for provider
- **GIVEN** a loaded provider with `evaluator-id: io.complytime.opa` but no cached complypack for that evaluator-id
- **WHEN** the scan pipeline invokes Generate
- **THEN** the `GenerateRequest` SHALL be sent without a complypack content path (backward compatible -- existing providers continue to work)

### Requirement: GenerateRequest extended with complypack content path
The `GenerateRequest` protobuf message SHALL include an optional `complypack_content_path` string field. When a complypack is available for the target evaluator-id, the field SHALL contain the absolute filesystem path to the cached complypack content. When no complypack is available, the field SHALL be empty. This is an additive wire-compatible change.

#### Scenario: Provider receives complypack path
- **GIVEN** a cached complypack for `io.complytime.opa`
- **WHEN** `RouteGenerate` dispatches to the opa provider
- **THEN** the `GenerateRequest.complypack_content_path` field SHALL contain the absolute path to the cached content file

#### Scenario: Provider receives empty complypack path
- **GIVEN** no cached complypack for the target evaluator-id
- **WHEN** `RouteGenerate` dispatches to the provider
- **THEN** the `GenerateRequest.complypack_content_path` field SHALL be empty

### Requirement: Providers command shows complypack availability
The `complyctl providers` command SHALL display which complypacks are cached for each discovered provider, based on evaluator-id matching. For each provider, the output SHALL show the complypack version if available, or indicate no complypack is cached.

#### Scenario: Provider with matched complypack
- **GIVEN** a provider with `evaluator-id: io.complytime.opa` and a cached complypack with matching evaluator-id version `1.0.0`
- **WHEN** the user runs `complyctl providers`
- **THEN** the output SHALL show complypack version `1.0.0` for that provider

#### Scenario: Provider without complypack
- **GIVEN** a provider with `evaluator-id: io.complytime.cel` and no cached complypack for that evaluator-id
- **WHEN** the user runs `complyctl providers`
- **THEN** the output SHALL indicate no complypack is available for that provider

### Requirement: Doctor checks complypack availability
The `complyctl doctor` command SHALL verify that each policy's evaluator-id has a matching complypack cached locally, using the `PolicyGraphResolver` to resolve evaluator-ids from policies. When a complypack is missing for a configured policy's evaluator-id, the doctor output SHALL emit a warning. This check SHALL only run when the workspace config contains a `complypacks` section.

#### Scenario: All complypacks present
- **GIVEN** a `complytime.yaml` with a `complypacks` section and all configured policies have matching cached complypacks
- **WHEN** the user runs `complyctl doctor`
- **THEN** the complypack check SHALL pass with no warnings

#### Scenario: Missing complypack
- **GIVEN** a `complytime.yaml` with a `complypacks` section and a configured policy with evaluator-id `io.complytime.opa` and no cached complypack for that evaluator-id
- **WHEN** the user runs `complyctl doctor`
- **THEN** the doctor output SHALL emit a warning indicating the missing complypack and suggest running `complyctl get`

#### Scenario: No complypacks section in config
- **GIVEN** a `complytime.yaml` without a `complypacks` section
- **WHEN** the user runs `complyctl doctor`
- **THEN** the complypack availability check SHALL be skipped entirely (no warnings emitted)

## MODIFIED Requirements

None -- all changes are additions.

## REMOVED Requirements

None -- this is purely additive.
