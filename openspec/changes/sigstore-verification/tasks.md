## 1. Dependencies and Configuration

- [x] 1.1 Add `sigstore-go` to `go.mod` and run `make vendor`
- [x] 1.2 Add `VerificationConfig` struct to `internal/complytime/` with `Issuer`, `Identity`, and `Key` fields; parse from `verification:` section in `complytime.yaml`
- [x] 1.3 Add validation: `Key` and `Issuer`/`Identity` are mutually exclusive; return error when both are set
- [x] 1.4 Write unit tests for verification config parsing and validation (valid keyless, valid keyed, invalid both, missing section)

## 2. Verification Module

- [x] 2.1 Create `internal/cache/verify.go` with `VerifyFunc` type and `VerificationResult` struct (Verified, SignerIdentity, Issuer, VerifiedAt)
- [x] 2.2 Implement `NewKeylessVerifier(issuer, identity string) VerifyFunc` using sigstore-go's `verify.NewVerifier` with `root.NewLiveTrustedRoot()`, `NewShortCertificateIdentity`, and `WithoutArtifactUnsafe()`
- [x] 2.3 Implement `NewKeyedVerifier(keyPath string) VerifyFunc` using sigstore-go's keyed verification with a PEM-encoded public key
- [x] 2.4 Write unit tests for `VerifyFunc` with mock sigstore entities (test both success and failure paths)

## 3. State Schema Extension

- [ ] 3.1 Add `Verified bool`, `SignerIdentity string`, `Issuer string`, and `VerifiedAt time.Time` fields to `PolicyState` with `json:",omitempty"` tags
- [ ] 3.2 Add `UpdatePolicyStateWithVerification` method (or extend existing `UpdatePolicyState`) to accept `VerificationResult`
- [ ] 3.3 Write unit tests for state serialization backward compatibility (unmarshal old format, marshal new format with omitempty)

## 4. Sync Pipeline Integration

- [ ] 4.1 Add `SyncOption` type and `WithVerifier(VerifyFunc) SyncOption` functional option to `Sync`
- [ ] 4.2 Modify `NewSync` to accept variadic `SyncOption` args and store the `VerifyFunc` on the `Sync` struct
- [ ] 4.3 Insert pre-copy verification step in `SyncPolicy`: after `DefinitionVersion()` resolves the remote reference, call `VerifyFunc` with the registry reference; skip `CopyPolicy` on failure
- [ ] 4.4 On successful verification, pass `VerificationResult` to state update; on skip (no verifier), set `Verified: false`
- [ ] 4.5 Add `SyncOption` and `WithVerifier` to `ComplypackSync` following the same pattern
- [ ] 4.6 Insert pre-copy verification in `SyncComplypack` pipeline: verify before `CopyComplypack`; suppress unverified warning when verification succeeds
- [ ] 4.7 Write unit tests for sync with verification (mock verifier success, failure, nil verifier)

## 5. CLI Integration

- [ ] 5.1 Add `--skip-verify` flag to `complyctl get` command in `get.go`
- [ ] 5.2 Read `VerificationConfig` from `WorkspaceConfig` and construct appropriate `VerifyFunc` (keyless or keyed); pass as `WithVerifier()` option to `Sync`/`ComplypackSync`
- [ ] 5.3 When `--skip-verify` is set or no verification config exists, do not pass a verifier; emit unverified warning for fetched artifacts
- [ ] 5.4 Add VERIFIED column to `complyctl list` output table, reading from `PolicyState.Verified`
- [ ] 5.5 Add verification status check to `complyctl doctor`: count verified vs unverified artifacts, warn on unverified
- [ ] 5.6 Write unit tests for CLI flag parsing and list output formatting

## 6. Governance Updates

- [ ] 6.1 Update THR02 mitigations in governance artifacts to reflect CTRL01.AR01 implementation status
- [ ] 6.2 Verify behavioral test `SignatureVerified()` expectations align with implementation

## 7. Integration Testing

- [ ] 7.1 Add E2E test for `complyctl get` with `--skip-verify` flag (verifies flag is accepted and warning is emitted)
- [ ] 7.2 Add E2E test for `complyctl list` VERIFIED column output
- [ ] 7.3 Run `make test-unit` and `make lint` to verify all tests pass and no lint violations
