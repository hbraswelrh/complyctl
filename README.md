# complytime

[![OpenSSF Best Practices status](https://www.bestpractices.dev/projects/9761/badge)](https://www.bestpractices.dev/projects/9761)
[![GoDoc](https://img.shields.io/static/v1?label=godoc&message=reference&color=blue)](https://pkg.go.dev/github.com/complytime/complytime)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/complytime/complytime/badge)](https://scorecard.dev/viewer/?uri=github.com/complytime/complytime)

ComplyTime leverages [OSCAL](https://github.com/usnistgov/OSCAL/) to perform compliance assessment activities, using plugins for each stage of the lifecycle.

## Documentation

:paperclip: [Installation](./docs/INSTALLATION.md)\
:paperclip: [Quick Start](./docs/QUICK_START.md)\
:paperclip: [Sample Component Definition](./docs/samples/sample-component-definition.json)

### Basic Usage

Determine the baseline you want to run a scan for and create an OSCAL [Assessment Plan](https://pages.nist.gov/OSCAL/learn/concepts/layer/assessment/assessment-plan/). The Assessment
Plan will act as configuration to guide the ComplyTime generation and scanning operations.

```bash
complytime list
...
# Table appears with options. Look at the Framework ID column.
```

```bash
complytime plan <framework-id>
...
# The file will be written out to assessment-plan.json in the specified workspace.
# Defaults to current working directory.

cat assessment-plan.json

complytime plan <framework-id> --dry-run

# The default assessment-plan.json contents will be written to standard output.
# Updates to the assessment-plan.json controls will be present in the updated configuration.
```

Run the generate command to `generate` policy artifacts in the workspace and run the `scan` command to execute the generated artifacts and get results.

```bash
complytime generate
...
complytime scan

# The results will be written to assessment-results.json in the specified workspace.
# Defaults to current working directory under folder "complytime".

complytime scan --with-md

# Both assessment-results.md and assessment-results.json will be written in the specified workspace.
# Defaults to current working directory under folder "complytime".
```

## Contributing

:paperclip: Read the [contributing guidelines](./docs/CONTRIBUTING.md)\
:paperclip: Read the [style guide](./docs/STYLE_GUIDE.md)\
:paperclip: Read and agree to the [Code of Conduct](./docs/CODE_OF_CONDUCT.md)

*Interested in writing a plugin?* See the [plugin guide](./docs/PLUGIN_GUIDE.md).
