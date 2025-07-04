# Quick Start

To get started with the `complyctl` CLI, at least one plugin must be installed with a corresponding OSCAL [Component Definition](https://pages.nist.gov/OSCAL/resources/concepts/layer/implementation/component-definition/).

> Note: Some of these steps are manual. The [quick_start.sh](../scripts/quick_start/quick_start.sh) automates the process below.

## Step 1: Install Complyctl

See [INSTALLATION.md](INSTALLATION.md)

## Step 2: Add configuration

After running `complyctl list` for the first time, the complytime
directory should be created under $HOME/.local/share

```markdown
complytime
├── bundles
└── plugins
└── controls
```

You will need an OSCAL Component Definition that defines an OSCAL Component for your target system and an OSCAL Component the corresponding
policy validation plugin. See `docs/samples/` for example configuration for the `myplugin` plugin.

```bash
cp docs/samples/sample-component-definition.json ~/.local/share/complytime/bundles
cp docs/samples/sample-profile.json docs/samples/sample-catalog.json ~/.local/share/complytime/controls
```

## Step 3: Install a plugin

Each plugin requires a plugin manifest. For more information about plugin discovery see [PLUGIN_GUIDE.md](PLUGIN_GUIDE.md).

```bash
cp bin/openscap-plugin ~/.local/share/complytime/plugins
checksum=$(sha256sum ~/.local/share/complytime/plugins/openscap-plugin| cut -d ' ' -f 1 )
cat > ~/.local/share/complytime/plugins/c2p-openscap-manifest.json << EOF
{
  "metadata": {
    "id": "openscap",
    "description": "My openscap plugin",
    "version": "0.0.1",
    "types": [
      "pvp"
    ]
  },
  "executablePath": "openscap-plugin",
  "sha256": "$checksum",
  "configuration": [
    {
      "name": "workspace",
      "description": "Directory for writing plugin artifacts",
      "required": true
    },
    {
      "name": "profile",
      "description": "The OpenSCAP profile to run for assessment",
      "required": true
    },
    {
      "name": "datastream",
      "description": "The OpenSCAP datastream to use. If not set, the plugin will try to determine it based on system information",
      "required": false
    },
    {
      "name": "policy",
      "description": "The name of the generated tailoring file",
      "default": "tailoring_policy.xml",
      "required": false
    },
    {
      "name": "arf",
      "description": "The name of the generated ARF file",
      "default": "arf.xml",
      "required": false
    },
    {
      "name": "results",
      "description": "The name of the generated results file",
      "default": "results.xml",
      "required": false
    }
  ]
}
EOF
```

## Step 4: Edit plugin configuration (optional)
```bash
mkdir -p /etc/complyctl/config.d
cp ~/.local/share/complytime/plugins/c2p-openscap-manifest.json /etc/complyctl/config.d
```

Edit `/etc/complyctl/config.d/c2p-openscap-manifest.json` to keep only the desired changes. e.g.:
```json
{
  "configuration": [
    {
      "name": "policy",
      "default": "custom_tailoring_policy.xml",
    },
    {
      "name": "arf",
      "default": "custom_arf.xml",
    },
    {
      "name": "results",
      "default": "custom_results.xml",
    }
  ]
}
```

### Using with the openscap-plugin

If using the openscap-plugin, there are two prerequisites:
- **openscap-scanner** package installed
- **scap-security-guide** package installed
