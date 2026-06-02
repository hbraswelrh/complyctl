## Context

`complyctl get` syncs policies from OCI registries into a local cache at `~/.complytime/policies/{id}/` as OCI Layout directories. The sync pipeline is: `complytime.yaml` -> `ParsePolicyRef` -> `registry.NewClient` -> `cache.NewRegistrySource` -> `cache.NewSync` -> `oras.Copy` (remote-to-local). The downstream commands (`generate`, `scan`) operate entirely from the local OCI Layout cache via `oci.New()` and never re-contact the registry.

The `PolicySource` interface (`internal/cache/source.go`) has only two methods (`DefinitionVersion`, `CopyPolicy`) and a single production implementation (`RegistrySource`). The test infrastructure (`MockPolicySource`, `MockBundlePolicySource` in `internal/cache/cachetest/`) already demonstrates pushing content into `*oci.Store` without a network, proving the interface supports local sources.

`ValidateOCIRef` (`internal/complytime/config.go`) is called at config load time by `get`, `generate`, and `scan`. It rejects URLs without a registry host containing `.` or `:`. This is the primary gate preventing local file paths.

## Goals / Non-Goals

**Goals:**
- Enable `complyctl get` to sync from a local OCI Layout directory via `oci-layout://` scheme
- Preserve full backward compatibility with existing registry-based behavior
- Use `oras.Copy()` for local-to-local transfer so incremental sync, digest verification, and state tracking all work identically to remote sources
- Require no changes to `generate`, `scan`, or the policy loader

**Non-Goals:**
- Supporting arbitrary file paths or directories that are not valid OCI Layouts (the source must be a proper OCI Layout with `oci-layout`, `index.json`, and `blobs/`)
- Adding a general-purpose `file://` scheme (too ambiguous about expected format)
- Modifying the mock OCI registry or devcontainer setup in this change (follow-up work)
- Supporting writing/publishing to local layouts (read-only source)

## Decisions

### 1. Use `oci-layout://` as the URL scheme

Use `oci-layout:///path/to/layout` as the URL format in `complytime.yaml`. The triple slash follows URI convention for local paths (scheme + empty authority + absolute path).

**Alternatives considered:**
- `file:///path` -- Too generic; does not communicate that the path must be a valid OCI Layout. Could be confused with raw YAML files.
- `oci:/path` -- Non-standard, could conflict with future OCI specification schemes.
- Bare paths (`/tmp/my-bundle`) -- Rejected by `ValidateOCIRef` host validation and ambiguous about expected directory structure.

### 2. Sentinel registry value for dispatch

`ParsePolicyRef` sets `ref.Registry = "oci-layout://"` for local sources. `syncSinglePolicy` checks this value to dispatch to `LocalSource` instead of `RegistrySource`. This avoids adding a new field to `PolicyRef` and keeps the dispatch logic minimal.

**Alternatives considered:**
- Add a `Scheme` field to `PolicyRef` -- Cleaner but touches more code for a single new scheme. Revisit if more schemes are added.
- Boolean `IsLocal` field -- Less extensible than the sentinel approach.

### 3. `LocalSource` uses `oras.Copy()` for local-to-local transfer

`LocalSource.CopyPolicy` opens the source OCI Layout with `oci.New()` and calls `oras.Copy(ctx, srcStore, tag, dstStore, tag, oras.CopyOptions{})`. This provides atomic transfer with digest verification, identical to the remote path.

**Alternatives considered:**
- Direct file copy (symlink or `os.Link`) -- Breaks digest verification, skips `oras` content-addressing guarantees, and couples the cache to the source directory lifecycle.
- Read manifest and push blobs individually -- Reimplements what `oras.Copy` already does.

### 4. Validate path existence at sync time, not config parse time

`ValidateOCIRef` accepts `oci-layout://` paths without checking if the path exists on disk. Path existence is validated when `LocalSource.CopyPolicy` calls `oci.New()` at sync time. This matches the existing pattern where registry reachability is not validated at config parse time.

**Alternatives considered:**
- Validate path existence in `ValidateOCIRef` -- Would make config loading side-effectful and fail for paths that will exist later (e.g., mounted volumes in containers).

### 5. Version handling defaults to `latest`

For `oci-layout://` sources without a `@version` suffix, the version defaults to `latest`. `LocalSource.DefinitionVersion` resolves the tag from the source layout's `index.json` via `store.Resolve(ctx, tag)`.

**Alternatives considered:**
- Require explicit version -- Adds friction for local use where there is typically only one version.

## Risks / Trade-offs

- **[Path portability]** `oci-layout://` paths are absolute and host-specific. A `complytime.yaml` with local paths is not portable across machines. Mitigation: this is the intended use case (private, local evaluation). Portable configs should use registry URLs.
- **[Source directory mutation]** If the source OCI Layout is modified after sync, `complyctl get` will detect the digest change on next run and re-sync. This is correct behavior but may surprise users who expect the cache to be independent. Mitigation: matches remote registry behavior exactly.
- **[No auth for local paths]** `LocalSource` does not use credentials. Filesystem permissions are the access control mechanism. Mitigation: appropriate for local files; registry auth remains for remote sources.
