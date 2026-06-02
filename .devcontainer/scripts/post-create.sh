#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# ---------------------------------------------------------------------------
# PATH setup — ensure built binaries and Go-installed tools are available
# ---------------------------------------------------------------------------
export PATH="./bin:${GOPATH:-$(go env GOPATH)}/bin:${PATH}"

# ---------------------------------------------------------------------------
# GITHUB_TOKEN least-privilege: capture and unset from environment
# Providers clone may need the token, but nothing else should see it.
# ---------------------------------------------------------------------------
_GITHUB_TOKEN="${GITHUB_TOKEN:-}"
unset GITHUB_TOKEN

if [[ -z "${_GITHUB_TOKEN}" ]]; then
    echo "WARNING: GITHUB_TOKEN was not set when the container started."
    echo "  complyctl scan requires a GitHub token to query branch"
    echo "  protection rules via snappy. get and generate work without it."
    echo "  Set it in your shell: export GITHUB_TOKEN=<your-token>"
else
    echo "NOTE: GITHUB_TOKEN was unset from the environment during setup"
    echo "  (least-privilege). Re-export it for scan commands:"
    echo "  export GITHUB_TOKEN=<your-token>"
fi

# ---------------------------------------------------------------------------
# Step 1: Build complyctl and mock-oci-registry
# make build compiles all cmd/ packages (complyctl, mock-oci-registry, etc.)
# ---------------------------------------------------------------------------
echo ">>> Building complyctl and mock-oci-registry..."
make build
echo "    Build complete. Binaries in ./bin/"

# ---------------------------------------------------------------------------
# Step 2: Install snappy, ampel, and conftest
#
# Pinned versions — update these when upgrading:
#   snappy    v0.2.4   https://github.com/carabiner-dev/snappy
#   ampel     v1.2.1   https://github.com/carabiner-dev/ampel
#   conftest  v0.68.2  https://github.com/open-policy-agent/conftest
# ---------------------------------------------------------------------------
echo ">>> Installing snappy, ampel, and conftest..."
go install github.com/carabiner-dev/snappy@v0.2.4
go install github.com/carabiner-dev/ampel/cmd/ampel@v1.2.1
go install github.com/open-policy-agent/conftest@v0.68.2
echo "    snappy, ampel, and conftest installed."

# ---------------------------------------------------------------------------
# Step 3: Clone and build complytime-providers (all providers)
# ---------------------------------------------------------------------------
echo ">>> Cloning complytime-providers..."
PROVIDERS_TMP="$(mktemp -d)"
trap 'rm -rf "${PROVIDERS_TMP}"' EXIT

# D6: Intentionally unpinned — tracks main for latest provider code.
# The commit SHA is logged below for auditability.
if ! git clone --depth 1 \
        https://github.com/complytime/complytime-providers.git \
        "${PROVIDERS_TMP}/complytime-providers"; then
    echo "FATAL: Failed to clone complytime-providers."
    echo "       This is an upstream dependency required for the dev environment."
    exit 1
fi

PROVIDERS_SHA="$(git -C "${PROVIDERS_TMP}/complytime-providers" \
    rev-parse HEAD)"
echo "    Cloned complytime-providers at ${PROVIDERS_SHA}"

echo ">>> Building complytime-providers..."
make -C "${PROVIDERS_TMP}/complytime-providers" build

echo ">>> Installing provider binaries..."
mkdir -p "${HOME}/.complytime/providers"
for provider in ampel openscap opa; do
    binary="complyctl-provider-${provider}"
    src="${PROVIDERS_TMP}/complytime-providers/bin/${binary}"
    if [[ -f "${src}" ]]; then
        cp "${src}" "${HOME}/.complytime/providers/"
        echo "    Installed ${binary}"
    else
        echo "    WARNING: ${binary} not found in build output, skipping."
    fi
done

# ---------------------------------------------------------------------------
# Step 4: Workspace setup — test-workspace with config and policies
# ---------------------------------------------------------------------------
echo ">>> Setting up test workspace..."
mkdir -p "${HOME}/test-workspace/.complytime/ampel/granular-policies"

cp tests/cross-repo/testdata/complytime.yaml \
    "${HOME}/test-workspace/"

cp tests/cross-repo/testdata/granular-policies/block-force-push.json \
    "${HOME}/test-workspace/.complytime/ampel/granular-policies/"

echo "    Test workspace ready at ~/test-workspace/"

# ---------------------------------------------------------------------------
# Step 4b: Pre-populate cache with local OCI Layout bundles (optional)
#
# If a bundles directory exists (default: /bundles/, override via
# COMPLYCTL_BUNDLES_DIR), discover OCI Layout bundles and copy them
# into the policy cache. This allows generate/scan to work with
# private bundles without complyctl get or any registry access.
#
# Uses jq (installed in Containerfile) for JSON manipulation.
# State.json schema must match internal/cache/state.go format.
# ---------------------------------------------------------------------------
BUNDLES_DIR="${COMPLYCTL_BUNDLES_DIR:-/bundles}"
if [[ -d "${BUNDLES_DIR}" ]]; then
    echo ">>> [optional] Pre-populating policy cache from ${BUNDLES_DIR}..."
    CACHE_DIR="${HOME}/.complytime"
    CACHE_POLICIES="${CACHE_DIR}/policies"
    STATE_FILE="${CACHE_DIR}/state.json"
    CONFIG_FILE="${HOME}/test-workspace/complytime.yaml"

    # Initialize state.json if it does not exist
    if [[ ! -f "${STATE_FILE}" ]]; then
        mkdir -p "${CACHE_DIR}"
        echo '{"last_sync":"0001-01-01T00:00:00Z","policies":{}}' \
            > "${STATE_FILE}"
    fi

    BUNDLE_COUNT=0
    for bundle_dir in "${BUNDLES_DIR}"/*/; do
        # Guard against glob matching nothing
        [[ -d "${bundle_dir}" ]] || continue

        bundle_name="$(basename "${bundle_dir}")"

        # Validate bundle name: only alphanumeric, hyphens, underscores
        if [[ ! "${bundle_name}" =~ ^[a-zA-Z0-9_-]+$ ]]; then
            echo "    Skipping ${bundle_name} (invalid characters in name)"
            continue
        fi

        # Skip if not a valid OCI Layout
        if [[ ! -f "${bundle_dir}/oci-layout" ]]; then
            echo "    Skipping ${bundle_name} (no oci-layout marker)"
            continue
        fi

        # Validate index.json before copying — ensures we don't leave
        # half-populated cache state on malformed bundles.
        manifest_digest=$(jq -r '.manifests[0].digest // empty' \
            "${bundle_dir}/index.json" 2>/dev/null || true)
        if [[ -z "${manifest_digest}" ]]; then
            echo "    Skipping ${bundle_name} (missing or malformed index.json)"
            continue
        fi

        # Copy OCI Layout to cache. The directory path under policies/
        # must match ref.Repository from ParsePolicyRef(url). For URL
        # localhost:0/policies/{name}, ref.Repository = "policies/{name}".
        dest="${CACHE_POLICIES}/policies/${bundle_name}"
        mkdir -p "${dest}"
        if ! cp -a "${bundle_dir}/." "${dest}/"; then
            echo "    WARNING: Failed to copy ${bundle_name}, skipping."
            continue
        fi
        echo "    Cached ${bundle_name}"

        # Update state.json with manifest digest using jq.
        # Uses a temp file + mv for atomic write to prevent corruption.
        now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
        policy_key="policies/${bundle_name}"
        if jq --arg key "${policy_key}" \
              --arg digest "${manifest_digest}" \
              --arg now "${now}" \
              '.policies[$key] = {
                  "version": "latest",
                  "digest": $digest,
                  "last_updated": $now
              } | .last_sync = $now' \
              "${STATE_FILE}" > "${STATE_FILE}.tmp"; then
            mv "${STATE_FILE}.tmp" "${STATE_FILE}"
            echo "    Updated state.json for ${policy_key}"
        else
            rm -f "${STATE_FILE}.tmp"
            echo "    WARNING: Failed to update state.json for ${bundle_name}"
        fi

        # Insert policy entry into complytime.yaml if not already present.
        # Uses localhost:0 as a dummy registry that passes ValidateOCIRef
        # (host contains ':') but is never contacted by generate/scan.
        if ! grep -q "localhost:0/policies/${bundle_name}" "${CONFIG_FILE}" 2>/dev/null; then
            # Insert before the first 'targets:' line to stay in policies block.
            if grep -q "^targets:" "${CONFIG_FILE}" 2>/dev/null; then
                # Use awk for reliable multi-line insertion across platforms.
                awk -v name="${bundle_name}" '
                    /^targets:/ {
                        print "  - url: localhost:0/policies/" name
                        print "    id: " name
                    }
                    { print }
                ' "${CONFIG_FILE}" > "${CONFIG_FILE}.tmp"
                mv "${CONFIG_FILE}.tmp" "${CONFIG_FILE}"
            else
                printf '  - url: localhost:0/policies/%s\n    id: %s\n' \
                    "${bundle_name}" "${bundle_name}" >> "${CONFIG_FILE}"
            fi
            echo "    Added ${bundle_name} to complytime.yaml"
        else
            echo "    ${bundle_name} already in complytime.yaml, skipping."
        fi

        BUNDLE_COUNT=$((BUNDLE_COUNT + 1))
    done

    if [[ ${BUNDLE_COUNT} -gt 0 ]]; then
        echo "    Pre-populated ${BUNDLE_COUNT} bundle(s)."
        echo "    NOTE: Use 'complyctl generate' and 'complyctl scan' directly."
        echo "    'complyctl get' will fail for local bundles (expected)."
    else
        echo "    No valid OCI Layout bundles found in ${BUNDLES_DIR}."
    fi
else
    echo ">>> No bundles directory at ${BUNDLES_DIR}, skipping cache"
    echo "    pre-population. Mount OCI Layout bundles at /bundles/ or"
    echo "    set COMPLYCTL_BUNDLES_DIR to enable."
fi

# ---------------------------------------------------------------------------
# Step 5: Start mock OCI registry
# ---------------------------------------------------------------------------
if curl -sf http://localhost:8765/v2/ > /dev/null 2>&1; then
    echo ">>> Mock OCI registry already running on port 8765."
else
    echo ">>> Starting mock OCI registry..."
    ./bin/mock-oci-registry &
    REGISTRY_PID=$!

RETRIES=0
MAX_RETRIES=30
until curl -sf http://localhost:8765/v2/ > /dev/null 2>&1; do
    RETRIES=$((RETRIES + 1))
    if [[ ${RETRIES} -ge ${MAX_RETRIES} ]]; then
        echo "FATAL: Mock OCI registry failed to start after ${MAX_RETRIES} retries."
        exit 1
    fi
    sleep 0.5
done

    echo "    Mock OCI registry running (PID: ${REGISTRY_PID}, port: 8765)"
fi

# ---------------------------------------------------------------------------
# Step 6: Record build commit for auto-rebuild detection
# ---------------------------------------------------------------------------
git rev-parse HEAD > ./bin/.build-commit

# ---------------------------------------------------------------------------
# Step 7: Persist PATH and auto-rebuild hook for interactive shells
# ---------------------------------------------------------------------------
REPO_ROOT="$(pwd)"
if ! grep -q "complyctl dev environment" "${HOME}/.bashrc" 2>/dev/null; then
    cat >> "${HOME}/.bashrc" << 'BASHRC'

# complyctl dev environment — added by post-create.sh
export PATH="REPO_ROOT_PLACEHOLDER/bin:${GOPATH:-$(go env GOPATH)}/bin:${PATH}"

# Auto-rebuild complyctl when source has changed (e.g., after
# checking out a PR branch). Skip with: export COMPLYCTL_SKIP_REBUILD=1
if [[ -z "${COMPLYCTL_SKIP_REBUILD:-}" ]]; then
    _repo="REPO_ROOT_PLACEHOLDER"
    _build_commit=""
    if [[ -f "${_repo}/bin/.build-commit" ]]; then
        _build_commit="$(cat "${_repo}/bin/.build-commit")"
    fi
    _head="$(git -C "${_repo}" rev-parse HEAD 2>/dev/null || true)"
    if [[ -n "${_head}" && "${_head}" != "${_build_commit}" ]]; then
        echo ">>> Source changed (${_build_commit:0:8}..${_head:0:8}), rebuilding complyctl..."
        if make -C "${_repo}" build 2>&1; then
            echo "${_head}" > "${_repo}/bin/.build-commit"
            echo "    Rebuild complete."
        else
            echo "    WARNING: Rebuild failed. Run 'make build' manually."
        fi
    fi
    unset _repo _build_commit _head
fi
BASHRC
    # Replace placeholder with actual repo root path
    sed -i "s|REPO_ROOT_PLACEHOLDER|${REPO_ROOT}|g" "${HOME}/.bashrc"
fi

echo ">>> Dev environment ready."
echo "    Test workspace: ~/test-workspace/"
echo "    Run: cd ~/test-workspace && complyctl get"
