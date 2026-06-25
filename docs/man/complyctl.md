% COMPLYCTL(1) Complyctl Manual
% Marcus Burghardt <maburgha@redhat.com>
% April 2025

# NAME

complyctl - Complyctl CLI performs compliance assessment activities using providers for different underlying technologies.

# SYNOPSIS

**complyctl** [command] [flags]

# DESCRIPTION

Complyctl is a lightweight compliance runtime that pulls Gemara policies from an OCI registry and executes scans via providers, producing compliance reports in multiple formats (EvaluationLog, OSCAL, SARIF, Markdown).

Complyctl can be extended to support desired policy engines by the use of providers. The provider acts as the integration between complyctl and the policy engine's native interface. Each provider is responsible for converting the policy content into the input format expected by the engine. In addition, the provider converts the raw results provided by the engine into the schema used by complyctl to generate output.

Providers communicate with complyctl via gRPC and can be authored using any preferred language. The provider acts as the gRPC server while the complyctl CLI acts as the client. When a complyctl command is run, it invokes the appropriate method served by the provider.

See more about authoring providers at https://github.com/complytime/complytime-providers/blob/main/docs/provider-guide.md.

Also check the complytime-providers repository at https://github.com/complytime/complytime-providers for provider-specific documentation.

# COMMANDS

**completion**
Generate the autocompletion script for the specified shell.

**doctor**
Run pre-flight diagnostics on the workspace. Checks provider discovery, policy
cache integrity, configuration validation, and complypack availability.

**generate**
Generate policy graph and invoke providers.

**get**
Fetch policies and complypacks from OCI registries into the local cache. When
complypacks are configured in complytime.yaml, their artifacts are fetched
alongside policies.

**help**
Display help about any command.

**init**
Initialize a complytime workspace with a complytime.yaml configuration file.

**list**
List information about available policies and targets.

**providers**
List discovered scanning providers and their health status. Includes a
COMPLYPACK column showing cached complypack versions per provider.

**scan**
Scan targets and produce compliance reports.

**version**
Print the version.

# OPTIONS

**-d**, **--debug**
Output debug logs.

**-h**, **--help**
Show help for complyctl.

Run **complyctl [command] --help** for more information about a specific command.

# EXAMPLES

## Initializing a workspace

```bash
$ complyctl init
# Creates a complytime.yaml configuration file in the current directory
```

## Fetching policies

```bash
$ complyctl get --policy-id my-policy
# Fetches the specified policy from the configured OCI registry

$ complyctl get
# Fetches all policies (and complypacks) configured in complytime.yaml
```

## Listing available policies and targets

```bash
$ complyctl list
# List all policies and targets in the workspace
```

## Running a scan

```bash
$ complyctl scan --policy-id my-policy
# Scan all targets for the specified policy

$ complyctl scan my-target
# Scan a specific target (policy-id inferred if target has exactly one policy)

$ complyctl scan my-target --policy-id my-policy --format pretty
# Scan a specific target with a specific policy, output as formatted Markdown
```

## Checking workspace health

```bash
$ complyctl doctor
# Run pre-flight diagnostics: provider discovery, cache integrity,
# configuration validation, and complypack availability
```

## Listing providers

```bash
$ complyctl providers
# List discovered providers, their health status, and cached complypack versions
```

# EXIT CODES

**0**
: Scan completed successfully. Findings (passed, failed, not applicable) are reported in the output but do not affect the exit code. Policy findings are data, not errors.

**1**
: An operational error occurred. This includes provider failures, invalid configuration, or zero requirements assessed. Reports and summaries are written before the non-zero exit so partial results remain available.

To gate a pipeline on compliance results, parse the scan output (SARIF, OSCAL) with your policy engine rather than relying on the exit code.

# SEE ALSO

See the upstream project at https://github.com/complytime/complyctl for more detailed documentation.

See the complytime-providers repository at https://github.com/complytime/complytime-providers for provider-specific documentation.

# COPYRIGHT

© 2025 Red Hat, Inc. Complyctl is released under the terms of the Apache-2.0 license.
