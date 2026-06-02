## Why

Private policy bundles cannot be committed to GitHub or embedded in the mock OCI registry testdata without exposing their content. The devcontainer environment currently only supports policies served by the mock registry (compile-time embedded) or fetched from a remote registry via `complyctl get`. Users evaluating private compliance policies in the devcontainer need the `generate` and `scan` pipeline to work with local bundles that never touch a registry or the repository.

## What Changes

- Extend the devcontainer post-create script (`.devcontainer/scripts/post-create.sh`) with a new step that discovers OCI Layout bundles from a configurable directory (`COMPLYCTL_BUNDLES_DIR`, default `/bundles/`) and pre-populates the `~/.complytime/policies/` cache
- For each discovered bundle, copy the OCI Layout into the cache directory, write a `state.json` entry with the manifest digest, and append a policy entry to the test workspace `complytime.yaml` using a dummy registry URL (`localhost:0/policies/{name}`) that passes `ValidateOCIRef` but is never contacted
- Add documentation for mounting private bundles into the devcontainer via DevPod volume mounts or manual copy
- No changes to `complyctl` source code -- this works with unmodified `complyctl` by pre-populating the cache that `generate` and `scan` read from

## Capabilities

### New Capabilities
- `bundle-cache-prepopulation`: Discover and pre-populate OCI Layout policy bundles into the devcontainer cache so that `complyctl generate` and `complyctl scan` work without `complyctl get` or any registry access

### Modified Capabilities

## Impact

- **Code**: `.devcontainer/scripts/post-create.sh` (new cache pre-population step)
- **Documentation**: `docs/TESTING_ENVIRONMENT.md` (document private bundle workflow)
- **Configuration**: `.devcontainer/devcontainer.json` (optional volume mount example)
- **Dependencies**: None -- uses existing `python3` (already in Fedora base image) for JSON manipulation
- **No complyctl code changes**: Relies entirely on the existing cache format that `generate` and `scan` already read from
