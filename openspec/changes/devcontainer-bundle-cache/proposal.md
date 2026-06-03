## Why

Private policy bundles cannot be committed to GitHub or embedded in the mock OCI registry testdata without exposing their content. The devcontainer environment currently only supports policies served by the mock registry (compile-time embedded) or fetched from a remote registry via `complyctl get`. Users evaluating private compliance policies in the devcontainer need the full `get` -> `generate` -> `scan` pipeline to work with local bundles that never touch a registry or the repository.

## What Changes

- Extend the mock OCI registry (`cmd/mock-oci-registry/main.go`) with a `seedFromDirectory()` method that discovers Gemara policy files from a mounted directory and serves them as OCI artifacts alongside the embedded testdata
- The post-create script (`.devcontainer/scripts/post-create.sh`) adds policy entries to `complytime.yaml` pointing at the mock registry (`http://localhost:8765/policies/{name}`), so `complyctl get` populates the cache through normal code paths
- Add documentation for mounting private bundles into the devcontainer via DevPod volume mounts or manual copy
- No changes to `complyctl` core source code -- this extends the mock registry (test infrastructure) to serve mounted policies

## Capabilities

### New Capabilities
- `bundle-cache-prepopulation`: Discover and serve mounted Gemara YAML policy files through the mock OCI registry so that the standard `complyctl get` -> `generate` -> `scan` workflow works for private policies without any registry access

### Modified Capabilities

## Impact

- **Code**: `cmd/mock-oci-registry/main.go` (`seedFromDirectory()`, `readFileLimited()`, security hardening), `cmd/mock-oci-registry/main_test.go` (12 tests)
- **Shell**: `.devcontainer/scripts/post-create.sh` (15-line registry-based config insertion replacing cache bypass)
- **Documentation**: `docs/TESTING_ENVIRONMENT.md` (Private Bundles section with raw YAML input format)
- **Configuration**: `.devcontainer/devcontainer.json` (optional volume mount example)
- **Dependencies**: None new -- reuses existing `addArtifact()` machinery in the mock registry
