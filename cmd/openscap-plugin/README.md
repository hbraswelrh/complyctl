# openscap-plugin

## Overview

NOTE: The development of this plugin is in progress and therefore it should only be used for testing purposes at this point.

**openscap-plugin** is a plugin which extends the **ComplyTime** capabilities to use OpenSCAP. The plugin communicates with **ComplyTime** via gRPC, providing a standard and consistent communication mechanism that gives independence for plugin developers to choose their preferred languages. This plugin is structured to allow modular development, ease of packaging, and maintainability.

For now, this plugin is developed together with ComplyTime for better collaboration during this phase of the project. In the future, this plugin may be decoupled into its own repository.

## Plugin Structure

```
openscap-plugin/
├── config/               # Package for plugin configuration
│ ├── config_test.go      # Tests for functions in config.go
│ └── config.go           # Main code used to process plugin configuration
├── oscap/                # Package to interact with oscap command
│ ├── oscap_test.go       # Tests for functions in oscap.go
│ └── oscap.go            # Main code used to interact with oscap command
├── scan/                 # Package to process system scan instructions
│ ├── scan_test.go        # Tests for functions in scan.go
│ └── scan.go             # Main code used to process scan instructions
├── server/               # Package to process server functions. Here is where the plugin communicates with ComplyTime CLI
│ ├── server_test.go      # Tests for functions in server.go
│ └── server.go           # Main code used to process server functions
├── xccdf/                # Package to process SCAP Datastreams
│ ├── datastream_test.go  # Tests for functions in datastream.go
│ ├── datastream.go       # Main code used to process Datastream files
│ ├── tailoring_test.go   # Tests for functions in tailoring.go
│ └── tailoring.go        # Main code used to generate tailoring files based on OSCAL and available Datastreams.
└── README.md             # This file
```

## Features

### Configuration

The plugin has some parameters that can be configured via the manifest file. Check the quick start [guide](../../docs/QUICK_START.md) to see an example.
ComplyTime process the manifest file and send the configuration values to the plugin.

These are the configuration used by openscap-plugin:
- **workspace**:  Directory used to read the tailoring file and to save oscap files generated during the scan. This configuration can also be set by ComplyTime CLI.
- **profile**:    Is the FrameworkID informed by ComplyTime. This FrameworkID corresponds to a profile ID in the Datastream.
- **datastream**: Datastream file to be used by `generate` and `scan` commands.
- **policy**:     File name for the tailoring file created by the `generate` command and consumed by the `scan` command.
- **arf**:        File name to save the `oscap` ARF results during the `scan` command.
- **results**:    File name to save `oscap` results during the `scan` command.

Note that the Datastream path is essential for the plugin commands and therefore a required option.
However it has no default value in the manifest because the plugin will try to determine the proper Datastream file automatically, based on system information. In case a Datastream file cannot be determined or validated, an error will be reported.
In exception cases, it is possible to manually define the desired Datastream path via manifest file.

### Generate

When the plugin receives the `generate` command from ComplyTime, it will use the informed Datastream and FrameworkID in combination with the `assessment-plan.json` file to:
* Process the `openscap` validation component from the `assessment-plan.json`
* Validate if all rules and variables in `assessment-plan.json` are valid in the Datastream
* Compare the rules, variables and variables values between the `assessment-plan.json` and the Datastream profile (FrameworkID)
* Generate a tailoring file to be used by the `scan` command
  * The tailoring file will extend the Datastream profile by overriding rules and variables values as defined in the `assessment-plan.json` file

### Scan
When the plugin receives the `scan` command from ComplyTime, it will use the informed Datastream and FrameworkID to:
* Validate the Datastream and Policy (tailoring file created by `generate` command) files.
* Assembly the `oscap` command
* Scan the system saving `oscap` results in ARF and results files according to the values defined in the plugin manifest file
* Process the results and return observations to ComplyTime so an `assessment-results.json` file can be created by `ComplyTime`

## Installation

### Prerequisites

- **Go** version 1.20 or higher
- **Make** (optional, for using the `Makefile` if included)
- **scap-security-guide** package installed

### Clone the repository

```bash
git clone https://github.com/complytime/complytime.git
cd complytime
```

## Build Instructions

To compile complytime and openscap-plugin:

```bash
make build
```

### Running

To use the plugin with `complytime`, see the quick start [guide](../../docs/QUICK_START.md).

### Testing

Tests are organized within each package. Whenever possible a unit test is created for every function.

Run tests using:

```bash
make test-units
```
