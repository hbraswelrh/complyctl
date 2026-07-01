## ADDED Requirements

### Requirement: Pre-copy signature verification
The system SHALL verify cosign/sigstore signatures on OCI artifacts via the
registry API before performing `oras.Copy()`. Unverified content SHALL NOT
be written to the local cache. Verification SHALL support both keyless
(OIDC issuer + SAN identity) and keyed (public key path) modes.

#### Scenario: Keyless verification succeeds
- **WHEN** `complyctl get` fetches a policy with a valid cosign signature
  signed via Sigstore keyless (Fulcio + Rekor) and the `verification:`
  section in `complytime.yaml` specifies a matching `issuer` and `identity`
- **THEN** the signature is verified before `oras.Copy()`, the policy is
  synced to the local cache, and `PolicyState.Verified` is set to `true`
  with signer identity and issuer recorded

#### Scenario: Keyless verification fails
- **WHEN** `complyctl get` fetches a policy whose cosign signature does not
  match the configured `issuer` or `identity`
- **THEN** `oras.Copy()` is NOT performed, the local cache is unchanged, and
  an error is returned describing the verification failure

#### Scenario: Keyed verification succeeds
- **WHEN** `complyctl get` fetches a policy with a valid cosign signature
  and the `verification:` section specifies a `key` path to a public key
  that matches the signing key
- **THEN** the signature is verified before `oras.Copy()`, the policy is
  synced to the local cache, and `PolicyState.Verified` is set to `true`

#### Scenario: No signature found with verification configured
- **WHEN** `complyctl get` fetches a policy that has no cosign signature
  attached and `verification:` is configured in `complytime.yaml`
- **THEN** the sync is aborted with an error describing the missing
  signature, and the local cache is unchanged. Use `--skip-verify` to
  bypass verification for unsigned artifacts

### Requirement: Verification bypass flag
The system SHALL provide a `--skip-verify` flag on `complyctl get` that
bypasses all signature verification. When `--skip-verify` is used, the
system SHALL emit the existing unverified warning message.

#### Scenario: Skip verification flag
- **WHEN** `complyctl get --skip-verify` is executed
- **THEN** no signature verification is performed, policies and complypacks
  are synced normally, and an unverified warning is emitted for each
  fetched artifact

### Requirement: Graceful degradation without verification config
The system SHALL silently skip verification when no `verification:` section
is present in `complytime.yaml`. A debug-level log message SHALL indicate
that verification was skipped due to missing configuration. This SHALL NOT
be a breaking change for existing users.

#### Scenario: No verification configured
- **WHEN** `complyctl get` is executed with a `complytime.yaml` that has no
  `verification:` section
- **THEN** policies and complypacks are synced without verification (existing
  behavior preserved), a debug-level log records "verification skipped: no
  verification configuration", and the unverified warning is emitted for
  fetched artifacts

### Requirement: Verification state persistence
The system SHALL persist verification metadata in `state.json` via the
`PolicyState` struct. The metadata SHALL include at minimum: verified status
(boolean), signer identity, OIDC issuer, and verification timestamp. All new
fields SHALL use `omitempty` for backward compatibility with existing state
files.

#### Scenario: State records verification success
- **WHEN** a policy is synced with successful signature verification
- **THEN** `state.json` contains `"verified": true`, `"signer_identity"`,
  `"issuer"`, and `"verified_at"` fields for that policy entry

#### Scenario: State backward compatibility
- **WHEN** a `state.json` file from a previous version (without verification
  fields) is loaded
- **THEN** all policies deserialize correctly with `Verified` defaulting to
  `false` and identity fields empty

### Requirement: Verification status in list output
The system SHALL display a VERIFIED column in `complyctl list` output showing
the verification status of each cached policy. The column SHALL show "Yes"
for verified policies, "No" for unverified policies, and "-" for policies
with no verification data.

#### Scenario: List shows verified status
- **WHEN** `complyctl list` is executed with a mix of verified and unverified
  policies in the cache
- **THEN** the output table includes a VERIFIED column with appropriate
  values for each policy

### Requirement: Doctor verification diagnostics
The system SHALL include a verification status check in `complyctl doctor`
output. The check SHALL report the count of verified vs unverified artifacts
and warn when cached artifacts lack verification.

#### Scenario: Doctor reports unverified artifacts
- **WHEN** `complyctl doctor` is executed and the cache contains unverified
  policies
- **THEN** the doctor output includes a warning about unverified cached
  policies with a count and recommendation to run `complyctl get` with
  verification configured

### Requirement: Complypack verification parity
The system SHALL apply the same verification pipeline to complypack artifacts
as to policy bundles. The `ComplypackSync` pipeline SHALL use the same
`VerifyFunc` mechanism and produce the same verification metadata in
`PolicyState` for complypack entries.

#### Scenario: Complypack verification succeeds
- **WHEN** `complyctl get` fetches a complypack with a valid cosign signature
  and verification is configured
- **THEN** the complypack signature is verified before `oras.Copy()`, the
  complypack is synced and unpacked, `PolicyState.Verified` is set to `true`,
  and the existing unverified warning is NOT emitted

#### Scenario: Complypack verification replaces warning
- **WHEN** `complyctl get` fetches a complypack and verification succeeds
- **THEN** the "WARNING: complypack has not been cryptographically verified"
  message is NOT emitted (verification succeeded, warning is unnecessary)

### Requirement: Verification configuration in complytime.yaml
The system SHALL support a `verification:` section in `complytime.yaml` for
configuring signature verification parameters. The section SHALL support
keyless verification via `issuer` and `identity` fields, and keyed
verification via a `key` field. The `key` and `issuer`/`identity` modes
SHALL be mutually exclusive.

#### Scenario: Keyless configuration parsed
- **WHEN** `complytime.yaml` contains a `verification:` section with `issuer`
  and `identity` fields
- **THEN** the verification pipeline uses keyless verification with the
  specified OIDC issuer and SAN identity

#### Scenario: Keyed configuration parsed
- **WHEN** `complytime.yaml` contains a `verification:` section with a `key`
  field pointing to a valid PEM-encoded public key
- **THEN** the verification pipeline uses keyed verification with the
  specified public key

#### Scenario: Invalid configuration rejected
- **WHEN** `complytime.yaml` contains a `verification:` section with both
  `key` and `issuer`/`identity` fields
- **THEN** configuration loading returns an error indicating that keyed and
  keyless verification are mutually exclusive
