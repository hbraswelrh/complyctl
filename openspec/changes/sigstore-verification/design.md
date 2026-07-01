## Context

complyctl fetches OCI policy bundles and complypack artifacts from registries
via `oras.Copy()` and stores them in a local cache. Neither the policy sync
pipeline (`internal/cache/sync.go`) nor the complypack sync pipeline
(`internal/cache/complypack_sync.go`) performs cryptographic signature
verification. THR02 identifies this as a supply chain risk, and CTRL01.AR01
formally requires signature verification before cache acceptance.

PR #641 shipped visibility improvements (unverified warnings, DIGEST column)
but deferred actual verification to this issue (#643). The `complypack`
library has `WithVerification`/`WithKeylessVerification` option stubs, but
they return "not yet implemented". Per the issue scope, verification logic
belongs in complyctl (single dependency, single code path for both artifact
types).

jpower432's comment recommends **pre-copy verification**: OCI signatures are
detached from the payload, so they can be verified via the registry API before
`oras.Copy()`. This prevents untrusted content from reaching disk and avoids
cache rollback complexity. The `go-containerregistry` dependency comes in
transitively with `sigstore-go`, so there is no extra client stack concern.

## Goals / Non-Goals

**Goals:**
- Add sigstore-go to verify cosign signatures on OCI artifacts before cache
  acceptance
- Support keyless verification (OIDC issuer + SAN identity) as the primary
  mode, matching how complyctl releases are signed
- Support keyed verification (public key path) as an alternative
- Implement pre-copy verification per jpower432's recommendation: verify via
  registry API before `oras.Copy()` so unverified content never touches disk
- Extend `PolicyState` with verification metadata from the sigstore-go
  `VerificationResult`
- Add `--skip-verify` flag to `complyctl get` for bypass scenarios
- Surface verification status in `complyctl list` (VERIFIED column) and
  `complyctl doctor` output
- Default to warn-only when no verification identity is configured (no
  breaking change for existing users)

**Non-Goals:**
- Signing artifacts (stays in complypack library)
- Enforcement mode (`--verify` that blocks sync on failure) -- follow-up
- Digest pinning -- orthogonal concern
- Modifying the `complypack` library's verification stubs

## Decisions

### 1. Pre-copy verification timing

**Decision**: Verify signatures via the registry API before calling
`oras.Copy()`. If verification fails, the copy is never performed and the
local cache remains unchanged.

**Rationale**: jpower432 recommended this approach because OCI cosign
signatures are stored as detached OCI objects (referrers or tagged artifacts).
sigstore-go's `cosign.VerifyImageSignatures` (or the lower-level
`verify.Verifier`) can resolve and verify these signatures directly from the
registry without pulling the payload. This means:
- Untrusted content never touches disk
- No cache rollback needed on verification failure
- The `go-containerregistry` remote client comes transitively via sigstore-go

**Alternatives considered**:
- Post-copy verification (verify from local OCI Layout after download):
  simpler integration but stores unverified content on disk, requiring cleanup
  on failure. Rejected per jpower432's guidance.

### 2. Verification module location

**Decision**: Create `internal/cache/verify.go` containing a `VerifyFunc`
type and a concrete sigstore-go implementation. Wire into sync pipelines via
functional options (`SyncOption`/`WithVerifier`).

**Rationale**: The cache package owns the sync pipeline and is the natural
integration point. A `VerifyFunc` type allows interface-based testing without
sigstore-go in test dependencies.

```go
// VerifyFunc verifies an OCI artifact identified by a registry reference.
// Returns a VerificationResult on success or an error on failure.
type VerifyFunc func(ctx context.Context, registryRef string) (*VerificationResult, error)

// VerificationResult captures sigstore-go verification output.
type VerificationResult struct {
    Verified       bool
    SignerIdentity string
    Issuer         string
    VerifiedAt     time.Time
}
```

### 3. Sync pipeline integration

**Decision**: Add `SyncOption` functional options to `Sync` and
`ComplypackSync`. When a `VerifyFunc` is set via `WithVerifier()`, it is
called after `DefinitionVersion()` resolves the remote digest but before
`CopyPolicy()`/`CopyComplypack()`. If verification fails, sync returns an
error without copying.

**Rationale**: Functional options preserve the existing `NewSync()` /
`NewComplypackSync()` signatures as backwards-compatible. The verification
step sits at the correct point in the pipeline -- after we know the remote
reference but before we transfer content.

### 4. State schema extension

**Decision**: Extend `PolicyState` with optional verification metadata fields:

```go
type PolicyState struct {
    Version        string    `json:"version"`
    Digest         string    `json:"digest"`
    EvaluatorID    string    `json:"evaluator_id,omitempty"`
    LastUpdated    time.Time `json:"last_updated"`
    Verified       bool      `json:"verified,omitempty"`
    SignerIdentity string    `json:"signer_identity,omitempty"`
    Issuer         string    `json:"issuer,omitempty"`
    VerifiedAt     time.Time `json:"verified_at,omitempty"`
}
```

**Rationale**: Using `omitempty` on all new fields ensures backward
compatibility -- existing `state.json` files without these fields deserialize
correctly with zero values. The fields capture the minimum useful verification
provenance: who signed (identity), which OIDC provider (issuer), and when
verification occurred. More detailed sigstore-go `VerificationResult` data
(Rekor entries, certificate chains) is logged at debug level but not persisted
to avoid schema bloat.

### 5. CLI integration

**Decision**:
- `complyctl get --skip-verify`: boolean flag (default false) that skips
  verification entirely. When skipped, the existing "WARNING: has not been
  cryptographically verified" message is emitted.
- `complyctl list`: add VERIFIED column showing Yes/No/- based on
  `PolicyState.Verified`.
- `complyctl doctor`: add a verification status check that reports whether
  cached artifacts have been verified, and warns on unverified artifacts.

**Rationale**: `--skip-verify` provides an escape hatch for broken signatures,
testing, and air-gapped environments. The VERIFIED column in `list` gives
users visibility into their cache integrity without running a separate
command.

### 6. Verification identity configuration

**Decision**: Verification identity (OIDC issuer + SAN) and keyed
verification (public key path) are configured via `complytime.yaml` under
a new `verification:` section:

```yaml
verification:
  # Keyless: OIDC issuer + subject identity
  issuer: "https://token.actions.githubusercontent.com"
  identity: "https://github.com/complytime/complyctl/.github/workflows/release.yml@refs/tags/*"
  # OR keyed: public key path
  # key: "/path/to/cosign.pub"
```

When no `verification:` section is present, verification is silently skipped
with a debug-level log. This preserves backward compatibility.

**Rationale**: Configuration in `complytime.yaml` keeps verification policy
alongside registry configuration. The issuer + identity pattern matches
cosign's `--certificate-oidc-issuer` + `--certificate-identity` flags, which
is the standard keyless verification interface.

### 7. Trusted root resolution

**Decision**: Use sigstore-go's `root.NewLiveTrustedRoot()` to fetch Sigstore's
public good instance trusted root via TUF. For keyed verification, construct
a trusted root from the provided public key.

**Rationale**: The live trusted root is the standard approach for verifying
artifacts signed via Sigstore's public infrastructure (Fulcio + Rekor). This
avoids bundling root certificates and handles key rotation automatically.

## Risks / Trade-offs

- **[Network dependency]** Pre-copy verification requires registry access for
  signature resolution and TUF root fetch. Air-gapped environments MUST use
  `--skip-verify`. -> Mitigation: `--skip-verify` flag with clear warning
  messaging.

- **[Transitive dependency size]** sigstore-go brings substantial transitive
  dependencies (go-containerregistry, protobuf, TUF, Rekor client). ->
  Mitigation: These are well-maintained, security-critical dependencies from
  the sigstore project. The dependency tree is auditable via `go mod graph`.

- **[State schema migration]** Adding fields to `PolicyState` changes the
  `state.json` format. -> Mitigation: All new fields use `omitempty`, so
  existing state files parse correctly. No migration needed.

- **[Pre-copy verification latency]** Signature verification adds a network
  round-trip before each copy. -> Mitigation: Signatures are small OCI
  objects; the latency is marginal compared to policy/complypack download
  times. The incremental sync check (digest comparison) still short-circuits
  when content hasn't changed.

- **[Unsigned artifacts]** Existing published artifacts may not have cosign
  signatures. -> Mitigation: When verification is configured but no signature
  is found, emit a warning and continue (warn-only mode). The `--skip-verify`
  flag provides a hard bypass.
