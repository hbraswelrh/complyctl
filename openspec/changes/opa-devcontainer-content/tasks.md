## 1. OPA Gemara Testdata

- [ ] 1.1 Create `cmd/mock-oci-registry/testdata/test-opa-catalog.yaml` with a minimal OPA-appropriate control catalog (e.g., workstation audit or container security controls with OPA-evaluable assessment requirements)
- [ ] 1.2 Create `cmd/mock-oci-registry/testdata/test-opa-policy.yaml` with `executor.id: opa` and assessment plans referencing the catalog requirements
- [ ] 1.3 Add `seedDefaults()` entry to seed `policies/test-opa-policy` with the OPA catalog and policy files, tagged `v1.0.0` and `latest`

## 2. OPA Complypack Artifact

- [ ] 2.1 Create minimal Rego policy file(s) that evaluate the assessment requirements defined in the OPA catalog
- [ ] 2.2 Create `complytime-mapping.json` mapping Gemara requirement IDs to Rego namespaces
- [ ] 2.3 Package Rego + mapping as a `content.tar.gz` payload embedded in the mock registry testdata
- [ ] 2.4 Add `seedDefaults()` entry to seed the OPA complypack via `addComplypackArtifact()` under `complypacks/test-opa-complypack` with evaluator-id `opa`

## 3. Workspace Configuration

- [ ] 3.1 Add OPA policy entry to `tests/cross-repo/testdata/complytime.yaml` with `id: test-opa-bp` pointing at `http://localhost:8765/policies/test-opa-policy`
- [ ] 3.2 Add `complypacks:` entry to `tests/cross-repo/testdata/complytime.yaml` pointing at `http://localhost:8765/complypacks/test-opa-complypack`
- [ ] 3.3 Add a target entry with variables appropriate for OPA evaluation (e.g., input file path or repository URL)

## 4. Post-create Script Updates

- [ ] 4.1 Update `.devcontainer/scripts/post-create.sh` to copy any OPA granular policies into the test workspace (if required by the OPA provider's Generate/Scan flow)
- [ ] 4.2 Verify the existing post-create.sh workspace setup copies the updated `complytime.yaml` with OPA entries

## 5. Documentation

- [ ] 5.1 Update `docs/TESTING_ENVIRONMENT.md` Command Reference section to include OPA provider commands alongside Ampel examples
- [ ] 5.2 Update `AGENTS.md` Recent Changes with the OPA devcontainer content entry

## 6. Validation

- [ ] 6.1 Verify `make build` compiles with the new embedded testdata
- [ ] 6.2 Verify `make test-unit` passes (mock registry tests)
- [ ] 6.3 Verify `make test-devcontainer` passes (Containerfile builds)
- [ ] 6.4 Verify `make lint` passes with zero issues
- [ ] 6.5 Manual test: start devcontainer, run `complyctl get`, verify both Ampel and OPA policies are fetched
- [ ] 6.6 Manual test: run `complyctl generate --policy-id test-opa-bp` and verify generation succeeds (requires OPA provider with `ComplypackContentPath` support)
- [ ] 6.7 Manual test: run `complyctl scan --policy-id test-opa-bp` and verify scan results are displayed
- [ ] 6.8 Manual test: verify existing `test-ampel-bp` workflow is unaffected
