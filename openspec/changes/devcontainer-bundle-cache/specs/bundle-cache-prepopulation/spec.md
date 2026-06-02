## ADDED Requirements

### Requirement: Discover bundles from a well-known directory
The post-create script SHALL discover OCI Layout bundles from a configurable directory. The default directory SHALL be `/bundles/`. The directory SHALL be configurable via the `COMPLYCTL_BUNDLES_DIR` environment variable. Each subdirectory containing an `oci-layout` marker file SHALL be treated as a valid bundle.

#### Scenario: Bundles directory exists with valid bundles
- **WHEN** the devcontainer starts with `/bundles/` containing one or more subdirectories with `oci-layout` marker files
- **THEN** the post-create script SHALL discover and process each bundle

#### Scenario: Bundles directory does not exist
- **WHEN** the devcontainer starts without a `/bundles/` directory
- **THEN** the post-create script SHALL skip bundle pre-population with an informational message and continue with remaining setup steps

#### Scenario: Custom bundles directory via environment variable
- **WHEN** `COMPLYCTL_BUNDLES_DIR` is set to `/custom/path`
- **AND** `/custom/path` contains bundle subdirectories
- **THEN** the post-create script SHALL discover bundles from `/custom/path` instead of `/bundles/`

#### Scenario: Subdirectory without oci-layout marker is skipped
- **WHEN** a subdirectory under the bundles directory does not contain an `oci-layout` file
- **THEN** the post-create script SHALL skip that subdirectory with an informational message

### Requirement: Pre-populate the OCI Layout cache
For each discovered bundle, the post-create script SHALL copy the OCI Layout into the `~/.complytime/policies/policies/{bundle-name}/` directory. The cache directory path SHALL match the `ref.Repository` value that `ParsePolicyRef` produces for the corresponding dummy URL.

#### Scenario: Bundle is copied to the correct cache path
- **WHEN** a bundle named `my-private-policy` is discovered
- **THEN** the post-create script SHALL copy the OCI Layout contents to `~/.complytime/policies/policies/my-private-policy/`
- **AND** the destination directory SHALL contain the `oci-layout` marker, `index.json`, and `blobs/` directory from the source

#### Scenario: Multiple bundles are cached
- **WHEN** the bundles directory contains `policy-a/` and `policy-b/` with valid OCI Layouts
- **THEN** both SHALL be copied to their respective cache directories

### Requirement: Update state.json with manifest digest
For each cached bundle, the post-create script SHALL extract the manifest digest from the bundle's `index.json` and write an entry to `~/.complytime/state.json`. The state entry key SHALL be `policies/{bundle-name}` to match the cache directory path.

#### Scenario: State file is created when it does not exist
- **WHEN** `~/.complytime/state.json` does not exist before bundle processing
- **THEN** the post-create script SHALL create it with an initial structure containing the bundle entry

#### Scenario: State file is updated when it already exists
- **WHEN** `~/.complytime/state.json` already exists (e.g., from a prior `complyctl get` of mock registry policies)
- **THEN** the post-create script SHALL add the bundle entry without overwriting existing entries

#### Scenario: Manifest digest is extracted correctly
- **WHEN** the bundle's `index.json` contains a manifest with digest `sha256:abc123`
- **THEN** the state entry SHALL have `"digest": "sha256:abc123"`

### Requirement: Append policy entries to workspace complytime.yaml
For each cached bundle, the post-create script SHALL append a policy entry to the test workspace `complytime.yaml`. The policy URL SHALL use the pattern `localhost:0/policies/{bundle-name}` which passes `ValidateOCIRef` validation but is never contacted by `generate` or `scan`.

#### Scenario: Policy entry is appended for a discovered bundle
- **WHEN** a bundle named `my-private-policy` is discovered and cached
- **THEN** the test workspace `complytime.yaml` SHALL contain a policy entry with `url: localhost:0/policies/my-private-policy` and `id: my-private-policy`

#### Scenario: Existing policies in complytime.yaml are preserved
- **WHEN** the test workspace `complytime.yaml` already contains the mock registry policy entry (`test-ampel-bp`)
- **AND** a local bundle is discovered
- **THEN** both the existing mock registry entry and the new bundle entry SHALL be present in the final `complytime.yaml`

### Requirement: Generate and scan work with pre-populated bundles
After cache pre-population, `complyctl generate` and `complyctl scan` SHALL work for pre-populated policies without running `complyctl get`.

#### Scenario: Generate succeeds for a pre-populated policy
- **WHEN** the cache contains a pre-populated bundle for `my-private-policy`
- **AND** `complytime.yaml` contains the corresponding policy and target entries
- **THEN** `complyctl generate --policy-id my-private-policy` SHALL resolve the policy from the cache and complete without error

#### Scenario: complyctl get fails for pre-populated policies
- **WHEN** the user runs `complyctl get` with a pre-populated policy URL `localhost:0/policies/my-private-policy`
- **THEN** the command SHALL fail because the dummy registry is not reachable
- **AND** this failure SHALL NOT affect the ability to run `generate` or `scan` using the pre-populated cache
