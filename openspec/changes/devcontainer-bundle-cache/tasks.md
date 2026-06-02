## 1. Bundle Discovery and Cache Pre-Population

- [x] 1.1 Add bundle discovery step to `.devcontainer/scripts/post-create.sh` between Step 4 (workspace setup) and Step 5 (mock registry): scan `COMPLYCTL_BUNDLES_DIR` (default `/bundles/`) for subdirectories with `oci-layout` marker files
- [x] 1.2 For each discovered bundle, copy the OCI Layout contents to `~/.complytime/policies/policies/{bundle-name}/`
- [x] 1.3 Skip subdirectories without `oci-layout` marker with an informational message
- [x] 1.4 If the bundles directory does not exist, print an informational message and continue

## 2. State File Management

- [x] 2.1 Initialize `~/.complytime/state.json` if it does not exist before processing bundles
- [x] 2.2 For each cached bundle, extract the manifest digest from the bundle's `index.json` using `python3`
- [x] 2.3 Update `state.json` with an entry keyed as `policies/{bundle-name}` containing the digest, version, and timestamp

## 3. Workspace Config Integration

- [x] 3.1 For each cached bundle, append a policy entry to `~/test-workspace/complytime.yaml` with URL `localhost:0/policies/{bundle-name}` and `id: {bundle-name}`
- [x] 3.2 Ensure existing mock registry policy entries in `complytime.yaml` are preserved

## 4. Documentation

- [x] 4.1 Add a "Private Bundles" section to `docs/TESTING_ENVIRONMENT.md` documenting: how to mount bundles via DevPod, the expected OCI Layout directory structure, the `COMPLYCTL_BUNDLES_DIR` env var, and the workflow (`generate`/`scan` without `get`)
- [x] 4.2 Add an optional volume mount example to `.devcontainer/devcontainer.json` (commented out) showing how to mount a local bundles directory

## 5. Validation

- [x] 5.1 Test manually: create a test OCI Layout bundle, mount at `/bundles/`, rebuild devcontainer, verify cache is populated and `complyctl list` shows the policy
- [x] 5.2 Verify `make test-devcontainer` still passes (Containerfile builds successfully)
- [x] 5.3 Verify existing mock registry workflow is unaffected: `complyctl get` + `complyctl generate` + `complyctl scan` still work for `test-ampel-bp`
