# Quick Start

To get started with the `complytime` CLI, at least one plugin must be installed with a corresponding OSCAL [Component Definition](https://pages.nist.gov/OSCAL/resources/concepts/layer/implementation/component-definition/).

> Note: Some of these steps are manual. The [quick_start.sh](../scripts/quick_start/quick_start.sh) automates the process below.

## Step 1: Install ComplyTime

See [INSTALLATION.md](INSTALLATION.md)

## Step 2: Add configuration

After running `complytime list` for the first time, the complytime
directory should be created under $HOME/.config

```markdown
complytime
├── bundles
└── plugins
└── controls
```

You will need an OSCAL Component Definition that defines an OSCAL Component for your target system and an OSCAL Component the corresponding
policy validation plugin. See `docs/samples/` for example configuration for the `myplugin` plugin.

```bash
cp docs/samples/sample-component-definition.json ~/.config/complytime/bundles
cp docs/samples/sample-profile.json docs/samples/sample-catalog.json ~/.config/complytime/controls
```

## Step 3: Install a plugin

Each plugin requires a plugin manifest. For more information about plugin discovery see [PLUGIN_GUIDE.md](PLUGIN_GUIDE.md).

```bash
cp bin/openscap-plugin ~/.config/complytime/plugins
checksum=$(sha256sum ~/.config/complytime/plugins/openscap-plugin| cut -d ' ' -f 1 )
cat > ~/.config/complytime/plugins/c2p-openscap-manifest.json << EOF
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

### Using with the openscap-plugin

If using the openscap-plugin, there are two prerequisites:
- **openscap-scanner** package installed
- **scap-security-guide** package installed
