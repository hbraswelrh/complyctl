% COMPLYTIME(1) ComplyTime Manual
% Marcus Burghardt <maburgha@redhat.com>
% April 2025

# NAME

complytime - ComplyTime CLI perform compliance assessment activities using plugins for different underlying technologies.

# SYNOPSIS

**complytime** [command] [flags]

# DESCRIPTION

ComplyTime CLI leverages OSCAL to perform compliance assessment activities, using plugins for each stage of the lifecycle.

ComplyTime can be extended to support desired policy engines (PVPs) by the use of plugins.
The plugin acts as the integration between ComplyTime and the PVPs native interface.
Each plugin is responsible for converting the policy content described in OSCAL into the input format expected by the PVP.
In addition, the plugin converts the raw results provided by the PVP into the schema used by ComplyTime to generate OSCAL output.

Plugins communicate with ComplyTime via gRPC and can be authored using any preferred language. The plugin acts as the gRPC server while the ComplyTime CLI acts as the client. When a complytime command is run, it invokes the appropriate method served by the plugin.

ComplyTime is built on https://github.com/oscal-compass/compliance-to-policy-go which provides a flexible plugin framework for leveraging OSCAL with various PVPs.

# COMMANDS

**completion**
Generate the autocompletion script for the specified shell.

**generate**
Generate PVP policy from an assessment plan.

**help**
Display help about any command.

**list**
List information about supported frameworks and components.

**plan**
Generate a new assessment plan for a given compliance framework ID.

**scan**
Scan environment with assessment plan.

**version**
Print the version.

# OPTIONS

**-d**, **--debug**
Output debug logs.

**-h**, **--help**
Show help for complytime.

Run **complytime [command] --help** for more information about a specific command.

# SEE ALSO

See the Upstream project at https://github.com/complytime/complytime for more detailed documentation.

See https://github.com/oscal-compass/compliance-to-policy-go project.

# COPYRIGHT

© 2025 Red Hat, Inc. ComplyTime is released under the terms of the Apache-2.0 license.
