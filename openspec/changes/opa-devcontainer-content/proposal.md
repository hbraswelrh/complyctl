## Why

The devcontainer environment currently supports only the Ampel
provider (`test-ampel-bp` policy-id). Users evaluating the OPA
provider have no ready-made test content in the devcontainer,
requiring manual setup of Gemara policies, Rego files, and
complypack configuration. Adding OPA test content to the
devcontainer enables end-to-end `get` -> `generate` -> `scan`
testing with the OPA provider alongside the existing Ampel
workflow.

## What Changes

- Add OPA-specific Gemara testdata (catalog.yaml + policy.yaml
  with `executor.id: opa`) to the mock registry's embedded
  testdata, seeded by `seedDefaults()`
- Seed an OPA complypack artifact in the mock registry containing
  Rego policies and `complytime-mapping.json`
- Update `complytime.yaml` in the test workspace to include the
  OPA policy-id and a `complypacks:` entry pointing at the mock
  registry's complypack artifact
- Update `post-create.sh` to copy OPA granular policies (if any)
  into the test workspace, mirroring the existing Ampel setup

## Capabilities

### New Capabilities
- `opa-test-content`: Embedded OPA Gemara testdata, complypack
  artifact, and workspace configuration for end-to-end OPA
  provider testing in the devcontainer

### Modified Capabilities

## Impact

- `cmd/mock-oci-registry/main.go`: New embedded testdata files
  and complypack artifact seeding in `seedDefaults()`
- `cmd/mock-oci-registry/testdata/`: New OPA catalog.yaml and
  policy.yaml files
- `tests/cross-repo/testdata/complytime.yaml`: Additional
  policy-id and complypacks entry
- `.devcontainer/scripts/post-create.sh`: OPA granular policy
  setup (if required by the OPA provider)
- Depends on PR #536 (complypack pull) for `complypacks:` config
  schema and `addComplypackArtifact()` in the mock registry
- Depends on a companion PR in `complytime-providers` for the OPA
  provider to consume `ComplypackContentPath`
