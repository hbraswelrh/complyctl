## ADDED Requirements

### Requirement: Mock registry serves OPA Gemara policy
The mock OCI registry SHALL serve an embedded OPA Gemara
policy (catalog + policy YAML) with `executor.id: opa` as
a split-layer OCI artifact under `policies/test-opa-policy`.

#### Scenario: OPA policy fetched via complyctl get
- **WHEN** the user runs `complyctl get` with a policy
  entry pointing at `localhost:8765/policies/test-opa-policy`
- **THEN** the OCI layout cache SHALL contain the OPA
  catalog and policy layers at
  `~/.complytime/policies/policies/test-opa-policy/`

### Requirement: Mock registry serves OPA complypack
The mock OCI registry SHALL serve an OPA complypack OCI
artifact containing Rego policy files and a
`complytime-mapping.json` file. The artifact SHALL use
the complypack artifact type and be tagged under
`complypacks/test-opa-complypack`.

#### Scenario: OPA complypack fetched via complyctl get
- **WHEN** the user runs `complyctl get` with a complypacks
  entry pointing at
  `localhost:8765/complypacks/test-opa-complypack`
- **THEN** the complypack cache SHALL contain the unpacked
  content at
  `~/.complytime/complypacks/{evaluator-id}/{version}/`

### Requirement: Test workspace configured for OPA provider
The test workspace `complytime.yaml` SHALL include a
policy entry for the OPA test policy and a complypacks
entry for the OPA complypack, both pointing at the mock
registry.

#### Scenario: OPA policy-id available after workspace setup
- **WHEN** the devcontainer starts and the post-create
  script completes
- **THEN** `complytime.yaml` in `~/test-workspace/` SHALL
  contain a policy entry with `id: test-opa-bp` pointing
  at the mock registry

#### Scenario: OPA complypack entry in workspace config
- **WHEN** the devcontainer starts and the post-create
  script completes
- **THEN** `complytime.yaml` in `~/test-workspace/` SHALL
  contain a complypacks entry pointing at the mock
  registry's OPA complypack artifact

### Requirement: End-to-end OPA pipeline works in devcontainer
The full `get` -> `generate` -> `scan` pipeline SHALL
succeed for the OPA provider in the devcontainer when the
OPA provider supports `ComplypackContentPath`.

#### Scenario: OPA generate succeeds
- **WHEN** the user runs
  `complyctl generate --policy-id test-opa-bp`
- **THEN** the command SHALL exit zero and produce
  generation artifacts for the OPA evaluator

#### Scenario: OPA scan produces results
- **WHEN** the user runs
  `complyctl scan --policy-id test-opa-bp`
- **THEN** the command SHALL exit zero and display
  requirement assessment results

### Requirement: Existing Ampel workflow unaffected
The existing `test-ampel-bp` policy-id and Ampel provider
workflow SHALL continue to work unchanged alongside the
new OPA content.

#### Scenario: Ampel pipeline still works
- **WHEN** the user runs `complyctl get` followed by
  `complyctl generate --policy-id test-ampel-bp` and
  `complyctl scan --policy-id test-ampel-bp`
- **THEN** all three commands SHALL succeed as before
