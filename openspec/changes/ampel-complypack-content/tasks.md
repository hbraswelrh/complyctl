## 1. Ampel Complypack Test Content

- [x] 1.1 [P] Create `cmd/mock-oci-registry/testdata/ampel-complypack/block-force-push.json` with valid `AmpelPolicy` content (copy from `tests/cross-repo/testdata/granular-policies/block-force-push.json`)
- [x] 1.2 Add `//go:embed testdata/ampel-complypack/*` directive and `ampelComplypackData` variable to `cmd/mock-oci-registry/main.go`, below the existing OPA embed directive
- [x] 1.3 Update `seedDefaults()` in `cmd/mock-oci-registry/main.go` to replace `buildDummyTarGz("policy.json", ...)` with `buildTarGzFromFS(ampelComplypackData, "testdata/ampel-complypack")` for the `complypacks/ampel-bp` artifact

## 2. Unit Tests

- [x] 2.1 Add `TestBuildTarGzFromFS_AmpelFS` test in `cmd/mock-oci-registry/main_test.go` to verify the ampel complypack archive contains `block-force-push.json` with valid content
- [x] 2.2 Update `TestSeedDefaults_AllReposSeeded` to verify the ampel complypack content blob decompresses to at least one `.json` file with a non-empty `id` field

## 3. Integration Test Cleanup

- [ ] 3.1 DEFERRED (merge ordering) Remove the `mkdir -p "${WORK_DIR}/.complytime/ampel/granular-policies"` and `cp` lines from `tests/cross-repo/cross_repo_integration_test.sh` (lines 176-179)
- [ ] 3.2 DEFERRED (merge ordering) Remove `tests/cross-repo/testdata/granular-policies/block-force-push.json` and its parent directory (content now embedded in mock registry)

## 4. Validation

- [x] 4.1 Verify `make build` compiles with the new embedded testdata
- [x] 4.2 Verify `make test-unit` passes (mock registry tests)
- [x] 4.3 Verify `make lint` passes with zero issues
