## Context

The devcontainer post-create script (`.devcontainer/scripts/post-create.sh`) currently builds `complyctl`, installs providers, starts a mock OCI registry on port 8765, and sets up a test workspace that syncs policies from the mock registry via `complyctl get`. This works for the embedded test policies but requires content to be compiled into the mock registry binary or fetched from a remote registry.

`complyctl generate` and `complyctl scan` read policies entirely from the local OCI Layout cache at `~/.complytime/policies/{repository}/` via `oci.New()` (`internal/policy/loader.go`). They never contact the registry. The cache directory name must match `ref.Repository` from `ParsePolicyRef(url)` -- for URL `localhost:0/policies/foo`, the directory is `policies/foo`. The `state.json` file tracks digest per policy for generation freshness but is optional -- if missing, `generate` always runs and `scan` always re-generates.

The `Validate()` function in `internal/complytime/config.go` calls `ValidateOCIRef()` on every policy URL at config load time, even in `generate` and `scan`. The URL must have a host containing `.` or `:` to pass validation, but it is never fetched by these commands.

## Goals / Non-Goals

**Goals:**
- Enable private OCI Layout bundles to be used in the devcontainer without committing them to the repository or pushing to a registry
- Pre-populate the `~/.complytime/policies/` cache so `generate` and `scan` work without `complyctl get`
- Make bundle discovery automatic via a well-known directory with an environment variable override
- Require no changes to `complyctl` source code

**Non-Goals:**
- Modifying `complyctl get` or `ValidateOCIRef` to support local file paths
- Supporting bundle hot-reload (bundles are copied at container creation time)
- Providing a bundle creation tool (users bring their own OCI Layouts)
- Changing the mock OCI registry behavior

## Decisions

### 1. Use a well-known directory with env var override for bundle discovery

Bundles are discovered from `/bundles/` by default, configurable via `COMPLYCTL_BUNDLES_DIR`. Each subdirectory containing an `oci-layout` marker file is treated as a bundle.

**Alternatives considered:**
- Explicit manifest file listing bundles -- Adds a configuration file that must be maintained. Auto-discovery from directory structure is simpler and self-documenting.
- Environment variable per bundle -- Does not scale. A single directory with named subdirectories is cleaner.

### 2. Use `localhost:0/policies/{name}` as the dummy URL in complytime.yaml

The `ValidateOCIRef` check requires URLs to have a host with `.` or `:`. Using `localhost:0` satisfies this (`:` present) while being clearly non-functional (port 0 is never bound). The `policies/` prefix ensures `ParsePolicyRef` produces `ref.Repository = "policies/{name}"`, matching the cache directory structure.

**Alternatives considered:**
- `example.com/policies/{name}` -- Works but could be confused with a real registry. `localhost:0` is unambiguous.
- `file:///path` -- Fails `ValidateOCIRef` and would require complyctl code changes.

### 3. Use python3 for JSON manipulation in the shell script

The post-create script needs to read `index.json` (extract manifest digest) and update `state.json` (add policy entries). Python3 is already available in the Fedora base image and handles JSON reliably. The alternative (`jq`) would require an additional package install.

**Alternatives considered:**
- `jq` -- Not installed by default. Adding it to the Containerfile is possible but adds a dependency for a single use.
- Pure bash with `grep`/`sed` -- Fragile for JSON manipulation. Not worth the risk.

### 4. Copy bundles into cache rather than symlink

Bundles are copied from `/bundles/` into `~/.complytime/policies/` rather than symlinked. This decouples the cache from the mount lifecycle and matches the behavior of `complyctl get` (which uses `oras.Copy` to create an independent OCI Layout in the cache).

**Alternatives considered:**
- Symlinks -- Would break if the mount is removed or changes. Introduces unexpected behavior if the source is modified after setup.
- Bind mount directly to cache path -- Fragile, couples container mount structure to internal cache layout.

## Risks / Trade-offs

- **[`complyctl get` will fail for pre-populated policies]** The dummy URL (`localhost:0/...`) is not a real registry, so `complyctl get` will error for these policies. Mitigation: this is expected and documented. Users run `generate` and `scan` directly. The mock registry policies (`test-ampel-bp`) still work via `get` as before.
- **[Cache format coupling]** The pre-population script depends on the OCI Layout cache format (`oci-layout`, `index.json`, `blobs/sha256/`). If `complyctl` changes its cache format, the script will break. Mitigation: the OCI Layout format is a standard; `complyctl` uses `oras-go` which enforces it. Format changes are unlikely and would break other things first.
- **[No incremental updates]** Bundles are copied once at container creation. If the source bundle is updated, the container must be rebuilt. Mitigation: acceptable for demo/test use. The post-create script runs on container creation, which is the natural rebuild trigger.
- **[Targets must be configured manually]** The script appends policy entries to `complytime.yaml` but cannot auto-generate target configurations (which depend on the specific repository being evaluated). Mitigation: document that users should edit the targets section of `complytime.yaml` after setup.
