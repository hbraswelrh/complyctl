## Context

The devcontainer post-create script (`.devcontainer/scripts/post-create.sh`) currently builds `complyctl`, installs providers, starts a mock OCI registry on port 8765, and sets up a test workspace that syncs policies from the mock registry via `complyctl get`. This works for the embedded test policies but requires content to be compiled into the mock registry binary or fetched from a remote registry.

The mock OCI registry already has all the machinery needed to serve additional policies. `seedPolicyFromFiles()` reads raw Gemara YAML files and `addArtifact()` wraps them into OCI artifacts with proper manifests, digests, and tags. The only difference between the embedded testdata and user-provided policies is the data source: `embed.FS` vs the filesystem.

## Goals / Non-Goals

**Goals:**
- Enable private Gemara YAML policy files to be served by the mock OCI registry in the devcontainer without committing them to the repository
- Provide a standard workflow where `complyctl get` -> `generate` -> `scan` works for all policies (both embedded and mounted)
- Make bundle discovery automatic via a well-known directory with an environment variable override
- Require no changes to `complyctl` core source code

**Non-Goals:**
- Modifying `complyctl get` or `ValidateOCIRef` to support local file paths
- Supporting bundle hot-reload (bundles are served from directory contents at registry startup)
- Providing a bundle creation tool (users bring their own Gemara YAML files)
- Supporting pre-built OCI Layout bundles as input (users provide raw `catalog.yaml` + `policy.yaml`)

## Decisions

### 1. Extend the mock registry to serve mounted policy files

The mock registry's `seedFromDirectory()` reads raw Gemara YAML files from a filesystem directory and registers them in the content store using the existing `addArtifact()` machinery -- exactly like the embedded testdata in `seedDefaults()`. This avoids coupling the shell script to cache internals (`state.json` schema, directory layout conventions) and ensures `complyctl get` works for all policies.

**Alternatives considered:**
- Direct cache pre-population via shell script -- Couples to `state.json` schema + directory layout. Breaks silently if cache internals change. Makes `complyctl get` fail for pre-populated policies. 125 lines of shell vs ~80 lines of Go.
- `jq`-based JSON manipulation -- Requires `jq` in the container. Fragile for managing OCI Layout digests.

### 2. Use `http://localhost:8765/policies/{name}` as the registry URL in complytime.yaml

Policy entries point at the mock registry's actual address with the `http://` scheme prefix. The `internal/registry/client.go` uses `strings.HasPrefix(registryURL, "http://")` to determine `plainHTTP` mode. Without the scheme prefix, the client defaults to HTTPS, which fails against the plainHTTP mock registry.

**Alternatives considered:**
- `localhost:0/policies/{name}` (dummy URL) -- Passes `ValidateOCIRef` but makes `complyctl get` fail. Creates a split workflow where some policies use `get` and some don't.
- `localhost:8765/policies/{name}` (no scheme) -- Fails because the registry client defaults to HTTPS without the `http://` prefix.

### 3. Use a well-known directory with env var override for bundle discovery

Bundles are discovered from `/bundles/` by default, configurable via `COMPLYCTL_BUNDLES_DIR` (shell side) / `MOCK_REGISTRY_CONTENT_DIR` (registry side). Each subdirectory containing `catalog.yaml` and `policy.yaml` is treated as a Gemara policy.

**Alternatives considered:**
- Explicit manifest file listing bundles -- Adds a configuration file that must be maintained. Auto-discovery from directory structure is simpler and self-documenting.
- Environment variable per bundle -- Does not scale. A single directory with named subdirectories is cleaner.

### 4. Security hardening for directory seeding

- **Name validation**: Directory names validated against `^[a-zA-Z0-9_-]+$` in both Go and shell (consistent)
- **Symlink rejection**: `entry.Type()` skips symlinked directories; `os.Lstat` rejects symlinked files
- **Resource exhaustion**: `readFileLimited()` caps file reads at 10 MB to prevent OOM from oversized files
- **No path traversal**: `os.ReadDir` returns base names only; paths constructed via `filepath.Join` with hardcoded filenames
- **Trust model**: The directory is operator-controlled (bind-mounted by the developer who owns the devcontainer)

## Risks / Trade-offs

- **[Cache format decoupling]** The registry-serving approach eliminates cache format coupling. `complyctl get` owns the cache population, so format changes in `state.json` or directory layout do not affect the script.
- **[No incremental updates]** Bundles are served from directory contents at registry startup. If the source files change, the registry must be restarted. Mitigation: acceptable for demo/test use.
- **[Split-layer format only]** `seedFromDirectory()` produces split-layer format artifacts (separate catalog + policy layers). If private policies need the Gemara bundle format (`DetectManifestShape()` in `internal/policy/loader.go`), a separate loading path would be needed. For standard Gemara catalog + policy pairs, the existing `addArtifact()` handles everything.
- **[Targets must be configured manually]** The script appends policy entries to `complytime.yaml` but cannot auto-generate target configurations (which depend on the specific repository being evaluated). Mitigation: document that users should edit the targets section of `complytime.yaml` after setup.
