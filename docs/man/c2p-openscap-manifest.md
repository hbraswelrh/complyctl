% C2P-OPENSCAP-MANIFEST.JSON(5) ComplyTime OpenSCAP Plugin Configuration
% Marcus Burghardt <maburgha@redhat.com>
% April 2025

# NAME

c2p-openscap-manifest.json - Configuration file for the OpenSCAP plugin used by ComplyTime

# DESCRIPTION

This file defines the metadata and runtime configuration options for the `openscap-plugin`, a plugin to be used with `complytime`.

It is a JSON-formatted file typically installed at:

**~/.config/complytime/plugins/c2p-openscap-manifest.json**

# FILE FORMAT

The configuration is a single JSON object with the following top-level keys:

- `metadata`: General plugin information
- `executablePath`: Name or path of the plugin binary
- `sha256`: The checksum of the binary (used for integrity checks)
- `configuration`: An array of runtime configuration options

# FIELDS

## metadata

```json
{
  "id": "openscap",
  "description": "My openscap plugin",
  "version": "0.0.1",
  "types": [ "pvp" ]
}
```

## executablePath

Path or name of the plugin binary to execute. Typically just:

```json
"executablePath": "openscap-plugin"
```

## sha256
SHA256 checksum of the plugin binary, used for runtime verification.

## configuration
A list of supported configuration parameters for the plugin.

Each entry includes:

- name: The name of the parameter
- description: Explanation of its purpose
- required: Whether this parameter must be provided
- default (optional): The default value if not specified

# CONFIGURATION OPTIONS
## workspace (required)
Directory for writing plugin artifacts.

## profile (required)
The OpenSCAP profile to run for assessment.

## datastream (optional)
The OpenSCAP datastream to use. If not set, the plugin will try to determine it based on system information.

## results (optional, default: results.xml)
The name of the generated results file.

## arf (optional, default: arf.xml)
The name of the generated ARF file.

## policy (optional, default: tailoring_policy.xml)
The name of the generated tailoring file.

# EXAMPLE
```json
{
  "workspace": "/tmp/scan",
  "profile": "xccdf_org.ssgproject.content_profile_cis",
  "datastream": "/usr/share/ssg-rhel9-ds.xml"
}
```

# SEE ALSO
complytime(1)

See the Upstream project at https://github.com/complytime/complytime for more detailed documentation.
