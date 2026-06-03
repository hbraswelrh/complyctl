## 1. Mock Registry Extension

- [x] 1.1 Add `seedFromDirectory()` method to `cmd/mock-oci-registry/main.go` that discovers Gemara policy directories from a filesystem path and registers them via `addArtifact()`
- [x] 1.2 Add `readFileLimited()` helper with symlink rejection and 10 MB size cap
- [x] 1.3 Add `validBundleName` regex (`^[a-zA-Z0-9_-]+$`) for directory name validation
- [x] 1.4 Add `resolveContentDir()` to read `MOCK_REGISTRY_CONTENT_DIR` env var (default `/bundles/`)
- [x] 1.5 Call `store.seedFromDirectory()` after `store.seedDefaults()` in `main()`

## 2. Post-Create Script Integration

- [x] 2.1 Add Step 4b to `.devcontainer/scripts/post-create.sh`: discover Gemara policy directories from `COMPLYCTL_BUNDLES_DIR` (default `/bundles/`)
- [x] 2.2 For each discovered bundle, append a policy entry to `complytime.yaml` with URL `http://localhost:8765/policies/{name}` and `id: {name}`
- [x] 2.3 Pass `MOCK_REGISTRY_CONTENT_DIR` to the mock registry process in Step 5
- [x] 2.4 Validate bundle names with the same `^[a-zA-Z0-9_-]+$` pattern used in Go
- [x] 2.5 Verify required Gemara YAML files (`catalog.yaml`, `policy.yaml`) exist before registering

## 3. Test Coverage

- [x] 3.1 Add unit tests for `resolveContentDir()` (default + env override)
- [x] 3.2 Add unit tests for `seedFromDirectory()` happy path (single and multiple policies)
- [x] 3.3 Add unit tests for skip conditions (missing catalog, missing policy, files instead of dirs, empty dir, nonexistent dir)
- [x] 3.4 Add security tests (invalid names, symlink directories, symlink files, oversized files)
- [x] 3.5 Add test for overwrite behavior when directory policy collides with embedded defaults

## 4. Documentation

- [x] 4.1 Add a "Private Bundles" section to `docs/TESTING_ENVIRONMENT.md` documenting: raw Gemara YAML input format, `COMPLYCTL_BUNDLES_DIR` env var, standard `get` -> `generate` -> `scan` workflow, and `seedFromDirectory()` explanation
- [x] 4.2 Add an optional volume mount example to `.devcontainer/devcontainer.json` (commented out) showing how to mount a local bundles directory

## 5. Validation

- [x] 5.1 Test manually: create test Gemara YAML files, mount at `/bundles/`, rebuild devcontainer, verify `complyctl get` fetches the policy and `complyctl list` shows it
- [x] 5.2 Verify `make test-devcontainer` still passes (Containerfile builds successfully)
- [x] 5.3 Verify existing mock registry workflow is unaffected: `complyctl get` + `complyctl generate` + `complyctl scan` still work for `test-ampel-bp`
