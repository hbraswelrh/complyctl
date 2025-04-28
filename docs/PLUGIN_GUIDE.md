# Plugin Authoring

ComplyTime can be extended to support desired policy engines (PVPs) by the use of plugins.
The plugin acts as the integration between ComplyTime and the PVPs native interface.
Each plugin is responsible for converting the policy content described in OSCAL into the input format expected by the PVP.
In addition, the plugin converts the raw results provided by the PVP into the schema used by ComplyTime to generate OSCAL output.

Plugins communicate with ComplyTime via gRPC and can be authored using any preferred language.
The plugin acts as the gRPC server while the ComplyTime CLI acts as the client.
When a `complytime` command is run, it invokes the appropriate method served by the plugin.

ComplyTime is built on [compliance-to-policy-go](https://github.com/oscal-compass/compliance-to-policy-go/ which provides a flexible plugin framework for leveraging OSCAL with various PVPs. For developers choosing Golang, the same SDK can be used for plugin authoring.

## Plugin Discovery

ComplyTime performs automated plugin discovery using the compliance-to-policy-go [plugin manager](https://github.com/complytime/compliance-to-policy-go/blob/CPLYTM-272/plugin/discovery.go).
Plugins are defined using manifest files placed in the `c2p-plugins` directory.
The plugin manifest is a JSON file that provides metadata about the plugin.
Check the quick start [guide](QUICK_START.md) to see an example.

**Note:** the plugin manifest file must have the following syntax for automatic discovery: `c2p-<plugin name>-manifest.json`

### Example Plugin Manifest

```
{
	“id”: “myplugin”,
	“description”: “my example plugin”,
	“version”: “0.1”,
	“type”: [“pvp”],
	“executablePath”: "myplugin" // in relation to the plugin directory
	“sha256”: “23f…” // sha256 of executable
	"configuration": [
      {
        "name": "config_name",
        "description": "Config description",
        "default": "default_value",
        "required": true
      },
	]
}
```

### Plugin Selection

ComplyTime generates a mapping of plugins to validation components at runtime.
This mapping uses the `title` of the validation component to find a matching plugin with that ID (defined in manifest).

```json
{
	...
	“uuid”: “701c7...”,
	“type”: “validation,
	“title”: “myplugin”, // name must match plugin ID in manifest
}
```

## Example

Below shows an example template for authoring a Golang plugin.

```go

import "github.com/oscal-compass/compliance-to-policy-go/v2/policy"

type PluginServer struct {}

func (s PluginServer) Generate(p policy.Policy) error {

	// PluginServer should implement the Generate() method to provide logic for
	// translating OSCAL to the PVPs expected input format.  Note: this may not be
	// applicable to all PVPs.

}

func (s PluginServer) GetResults(p policy.Policy) (policy.PVPResult, error) {

	// PluginServer should implement the GetResults() method to provide logic to
	// collect results from the PVP for a given policy.  Note: if the PVP requires input
	// from Generate() then the policy input here may be ignored.

}
```
