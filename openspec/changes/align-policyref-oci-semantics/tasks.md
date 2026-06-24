## 1. Deprecation Warning for @version Notation

- [x] 1.1 Add deprecation warning in `ParsePolicyRef` (`internal/complytime/config.go`) at the `@version` to `:tag` conversion point (line ~192). Emit to `os.Stderr` with format: `DEPRECATED: @version notation in policy URL "<raw>". Use ":tag" syntax instead. @version support will be removed in a future release.`
- [x] 1.2 Add unit tests in `internal/complytime/config_test.go` verifying: (a) full OCI ref with `@v1.0` emits deprecation warning and returns correct `Tag`; (b) full OCI ref with `@sha256:...` does not emit warning; (c) bare policy ID with `@v2.0` does not emit warning.

## 2. Doctor Error Reporting -- CheckPolicyActivePeriod

- [x] 2.1 Update `CheckPolicyActivePeriod` in `internal/doctor/doctor.go` to report `ParsePolicyRef` failures as `CheckResult{Status: StatusFail, Blocking: true}` with message `"invalid policy reference: <err>"` instead of silently continuing (lines 607-609).
- [x] 2.2 Add unit test in `internal/doctor/doctor_test.go` verifying that a malformed policy URL produces a `StatusFail` result with `Blocking: true`.

## 3. Doctor Error Reporting -- CheckComplypacks

- [x] 3.1 Update `CheckComplypacks` in `internal/doctor/doctor.go` to report `ParsePolicyRef` failures as `CheckResult{Status: StatusFail, Blocking: true}` with message `"invalid policy reference: <err>"` instead of silently continuing (lines 792-794).
- [x] 3.2 Add unit test in `internal/doctor/doctor_test.go` verifying that a malformed policy URL produces a `StatusFail` result with `Blocking: true`.

## 4. Doctor Error Reporting -- CheckVariables

- [x] 4.1 Replace `resolveFailures++` counter in `CheckVariables` (`internal/doctor/doctor.go`, lines 440-470) with per-failure `CheckResult{Status: StatusWarn, Blocking: false}` results. Cover four failure cases: (a) policy not found in config; (b) `ParsePolicyRef` error; (c) version resolution error; (d) graph resolution error. Each result message SHALL identify the policy and the specific error.
- [x] 4.2 Add unit tests in `internal/doctor/doctor_test.go` verifying each failure case produces a `StatusWarn` result with a descriptive message.

## 5. Documentation and Changelog

- [x] 5.1 Add deprecation notice to `CHANGELOG.md` under the Unreleased section noting the `@version` deprecation and recommended migration to `:tag` syntax.

## 6. Verification

- [x] 6.1 Run `make test-unit` and verify all existing and new tests pass.
- [x] 6.2 Run `make lint` and verify no linting violations.
