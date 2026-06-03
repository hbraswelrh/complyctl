# Changelog

## Unreleased

### Added

- Devcontainer configuration for interactive CLI testing during PR
  reviews (`.devcontainer/`, `docs/TESTING_ENVIRONMENT.md`,
  `make test-devcontainer`). Supports GitHub Codespaces, DevPod, and
  VS Code Dev Containers.
- Cross-repo integration test infrastructure validating the complyctl + Ampel
  provider pipeline end-to-end (`tests/cross-repo/`, `make test-cross-repo`).
- CI workflow `ci_cross_repo_integration.yml` that builds complyctl from the PR
  branch and complytime-providers from main, then runs the full get → generate →
  scan pipeline with real snappy and ampel binaries.
- Minimal test Gemara policy (`policies/test-branch-protection`) seeded in the
  mock OCI registry for integration testing.
- OPA provider test content in devcontainer: Gemara testdata (catalog +
  policy with `executor.id: opa`), OPA complypack artifact (Rego policies
  + `complytime-mapping.json`), `test-opa-bp` policy-id and
  `test-k8s-deployment` target in workspace configuration. OPA command
  reference added to `docs/TESTING_ENVIRONMENT.md`.
