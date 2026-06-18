## 1. ParsePolicyRef Rewrite

- [x] 1.1 Rewrite `ParsePolicyRef` in `internal/complytime/config.go` to delegate to `oras.land/oras-go/v2/registry.ParseReference` for inputs containing `/`, handling scheme stripping and bare IDs separately. Change signature to `(PolicyRef, error)`.
- [x] 1.2 Update `internal/complytime/config_test.go`: update existing 7 test cases for `(PolicyRef, error)` return and add new cases for `:tag`, `@sha256:digest`, empty input, and whitespace-only input.

## 2. Config Load Validation

- [x] 2.1 Add a validation loop in `LoadFrom` (`internal/complytime/config.go`) after YAML unmarshal to call `ParsePolicyRef` on all `Policies` and `Complypacks` entry URLs, returning an error on the first invalid reference.

## 3. Sync Lookup Reference Fix

- [x] 3.1 Extract a `buildLookupRef(repository, version string) string` helper (in `internal/cache/` or alongside the sync functions) that uses `@` for digest versions and `:` for tag versions.
- [x] 3.2 Update `SyncPolicy` in `internal/cache/sync.go` to use `buildLookupRef`.
- [x] 3.3 Update `SyncComplypack` in `internal/cache/complypack_sync.go` to use `buildLookupRef`.

## 4. Caller Updates

- [x] 4.1 Update `syncSinglePolicy` and `syncSingleComplypack` in `cmd/complyctl/cli/get.go` to handle `ParsePolicyRef` error return.
- [x] 4.2 Update `scanOptions.scanPolicy` in `cmd/complyctl/cli/scan.go` to handle `ParsePolicyRef` error return.
- [x] 4.3 Update list command in `cmd/complyctl/cli/list.go` to handle `ParsePolicyRef` error return.
- [x] 4.4 Update `generateOptions.generatePolicy` in `cmd/complyctl/cli/generate.go` to handle `ParsePolicyRef` error return.
- [x] 4.5 Update 4 call sites in `internal/doctor/doctor.go` (`CheckPolicyStaleness`, `CheckVariables`, `CheckPolicyExpiry`, `CheckComplypackCache`) to handle `ParsePolicyRef` error return.

## 5. Verification

- [x] 5.1 Run `make test-unit` and confirm all tests pass.
- [x] 5.2 Run `make lint` and confirm no lint errors.
