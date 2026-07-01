## Why

OCI policy bundles and complypack artifacts are fetched from registries and
accepted into the local cache without cryptographic signature verification.
THR02 (OCI Registry Artifact Tampering) identifies this as a supply chain risk,
and CTRL01 (Policy Content Signature Validation) formally requires verification
before cache acceptance. PR #641 shipped visibility improvements (unverified
warnings, DIGEST column) but deferred the actual sigstore-go verification to
be designed against the real API. jpower432 recommends pre-copy verification
since OCI signatures are detached and verifiable via registry API before
`oras.Copy`, avoiding untrusted content in the cache. This addresses
issue #643.

## What Changes

- Add `sigstore-go` dependency to `go.mod` for cosign/sigstore verification
- Create `internal/cache/verify.go` with a `VerifyFunc` type and sigstore-go
  verification implementation supporting both keyless (OIDC issuer + identity)
  and keyed (public key path) verification modes
- Implement pre-copy verification: verify signatures via registry API before
  `oras.Copy` so unverified content never touches disk
- Wire verification into both policy sync (`SyncPolicy`) and complypack sync
  (`SyncComplypack`) pipelines via `WithVerifier(...)` functional options
- Add `--skip-verify` flag to `complyctl get` for bypassing verification
- Extend `PolicyState` with verification metadata (verified status, signer
  identity, verification timestamp)
- Add VERIFIED column to `complyctl list` output
- Surface verification status in `complyctl doctor` diagnostics
- Update THR02 mitigations in governance artifacts
- Default behavior: when no verification identity/keys are configured, skip
  verification silently (debug log) with no breaking change for existing users

## Capabilities

### New Capabilities
- `artifact-verification`: Cryptographic signature verification for OCI
  artifacts (policies and complypacks) using sigstore-go, covering keyless
  and keyed verification modes, pre-copy timing, state persistence, and
  CLI integration

### Modified Capabilities

## Impact

- **Dependencies**: `sigstore-go` added to `go.mod`; transitive deps include
  `go-containerregistry`, protobuf types, TUF client, Rekor client
- **Code**: `internal/cache/` (new verify.go, modified sync.go,
  complypack_sync.go, state.go, source.go), `cmd/complyctl/cli/` (get.go,
  list.go, doctor.go)
- **APIs**: `PolicySource` and `ComplypackSource` interfaces unchanged;
  verification is orthogonal via functional options on `Sync`/`ComplypackSync`
- **Governance**: CTRL01.AR01 transitions from gap to implemented; THR02
  mitigations updated
- **Users**: No breaking changes; verification is opt-in via identity/key
  configuration and skippable via `--skip-verify`
