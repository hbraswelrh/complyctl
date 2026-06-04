## ADDED Requirements

### Requirement: Valid ampel complypack content in mock registry
The mock OCI registry SHALL seed the `complypacks/ampel-bp` artifact
with a tar.gz payload containing valid ampel granular policy JSON that
the ampel provider's `LoadGranularPolicies()` accepts without error.
Each embedded JSON file SHALL contain an `id` field, a `meta` object
with `description` and `controls`, and a `tenets` array with at least
one entry.

#### Scenario: Mock registry serves valid ampel complypack
- **WHEN** the mock registry starts with default seed data
- **THEN** the `complypacks/ampel-bp` artifact SHALL contain a tar.gz
  layer with at least one `.json` file that conforms to the `AmpelPolicy`
  schema (non-empty `id`, `meta.controls`, and `tenets` fields)

#### Scenario: Ampel provider parses complypack content without error
- **WHEN** the ampel provider's `Generate()` receives a
  `ComplypackContentPath` pointing to the extracted complypack content
- **THEN** `LoadGranularPolicies()` SHALL parse all JSON files
  successfully and return a non-empty policy map

### Requirement: Ampel complypack embedded via filesystem
The ampel complypack content SHALL be embedded using `//go:embed` and
`buildTarGzFromFS()`, matching the pattern established by the OPA
complypack. The content SHALL NOT use `buildDummyTarGz()` with inline
string literals.

#### Scenario: Embedded testdata directory structure
- **WHEN** the mock registry binary is compiled
- **THEN** the `testdata/ampel-complypack/` directory SHALL be embedded
  via `//go:embed` and its contents packaged by `buildTarGzFromFS()`

### Requirement: Cross-repo integration test uses complypack flow
The cross-repo integration test SHALL exercise the complypack content
path for the ampel provider. Manual pre-staging of granular policy files
into the workspace default directory SHALL be removed so the test
validates the complypack-delivered content path.

#### Scenario: Integration test passes without pre-staged granular policies
- **WHEN** the cross-repo integration test runs `complyctl generate`
- **THEN** the ampel provider SHALL load granular policies from the
  complypack content path (delivered by `complyctl get`) instead of from
  a manually pre-staged `.complytime/ampel/granular-policies/` directory

#### Scenario: Backward compatibility with old provider
- **WHEN** the mock registry serves valid ampel complypack content
- **AND** the ampel provider binary does NOT support `ComplypackContentPath`
  (pre-PR#52 version)
- **THEN** the provider SHALL ignore the complypack content and fall back
  to the pre-staged granular policies directory without error

### Requirement: Unit test coverage for ampel complypack seeding
The mock registry's unit tests SHALL verify that the ampel complypack
artifact is seeded with valid, readable content containing the expected
files.

#### Scenario: TestSeedDefaults verifies ampel complypack content
- **WHEN** `TestSeedDefaults_AllReposSeeded` runs
- **THEN** it SHALL verify that `complypacks/ampel-bp` contains a content
  blob that decompresses to at least one `.json` file with a non-empty
  `id` field
