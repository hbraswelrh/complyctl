## 1. Config Parsing: Accept oci-layout:// Scheme

- [x] 1.1 Add `oci-layout://` early return in `ValidateOCIRef` (`internal/complytime/config.go`): accept the scheme, reject empty path, preserve shell metachar check
- [x] 1.2 Add `oci-layout://` branch in `ParsePolicyRef` (`internal/complytime/config.go`): set `Registry = "oci-layout://"`, `Repository = path`, extract `@version` suffix
- [x] 1.3 Add unit tests for `ValidateOCIRef` with `oci-layout://` URLs (`internal/complytime/config_test.go`): valid path, empty path rejection, metachar rejection, version suffix
- [x] 1.4 Add unit tests for `ParsePolicyRef` with `oci-layout://` URLs (`internal/complytime/config_test.go`): path extraction, version extraction, no-version default

## 2. LocalSource Implementation

- [x] 2.1 Add `LocalSource` struct and `NewLocalSource` constructor in `internal/cache/source.go`
- [x] 2.2 Implement `LocalSource.DefinitionVersion`: open source layout with `oci.New()`, resolve tag via `store.Resolve()`, return digest and version
- [x] 2.3 Implement `LocalSource.CopyPolicy`: open source layout with `oci.New()`, call `oras.Copy()` from source store to destination store
- [x] 2.4 Add unit tests for `LocalSource` (`internal/cache/source_test.go`): create temp OCI Layout with test manifest and layer, verify `DefinitionVersion` returns correct digest and tag, verify `CopyPolicy` copies all content to destination store
- [x] 2.5 Add unit test for `LocalSource` with nonexistent path: verify meaningful error message
- [x] 2.6 Add unit test for `LocalSource` with invalid OCI Layout (missing `oci-layout` marker): verify error

## 3. Scheme Dispatch in complyctl get

- [x] 3.1 Add scheme dispatch in `syncSinglePolicy` (`cmd/complyctl/cli/get.go`): if `ref.Registry == "oci-layout://"`, create `LocalSource` instead of `RegistrySource`
- [x] 3.2 Skip `resolveLatestVersion` for local sources: local version resolution is handled by `LocalSource.DefinitionVersion`

## 4. Integration Testing

- [x] 4.1 Add E2E test for local OCI Layout sync (`tests/e2e/`): create temp OCI Layout with test policy, write `complytime.yaml` with `oci-layout://` URL, run `complyctl get`, verify cache is populated
- [x] 4.2 Add E2E test for mixed sources: config with both `oci-layout://` and registry-based URLs, verify both sync correctly
- [x] 4.3 Add E2E test for incremental sync: run `complyctl get` twice with unchanged local layout, verify second run skips sync

## 5. Validation

- [x] 5.1 Run `make test-unit` and confirm all tests pass
- [x] 5.2 Run `make lint` and `make vet` and confirm no issues
- [x] 5.3 Run `make test-e2e` and confirm E2E tests pass including new local layout tests
