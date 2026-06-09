## Why

The `complyctl scan` command produces `gemara.EvaluationLog` output, but the
per-step assessment data from providers is lost during conversion. Providers
send `provider.Step` structs (Name, Result, Message) via gRPC, and complyctl
correctly aggregates them into a top-level result and counts them in
`StepsExecuted`. However, the `gemara.AssessmentLog.Steps` field is never
populated because `provider.Step` (a data struct) must be wrapped into
`gemara.AssessmentStep` (a closure type). This causes evaluation log YAML to
always contain `steps: []`, which fails Gemara schema validation and prevents
downstream consumers from inspecting individual assessment steps.

## What Changes

- Add a conversion function that wraps each `provider.Step` into a
  `gemara.AssessmentStep` closure, capturing the step's Name, Result, and
  Message as fixed return values.
- Add a `providerResultToGemara` mapping function in `evaluator.go` (or
  reuse the existing one from `sarif.go`) to convert `provider.Result` to
  `gemara.Result` within the closure.
- Update `providerToGemaraAssessment()` in `internal/output/evaluator.go`
  to populate the `Steps` field on the returned `gemara.AssessmentLog`.
- Add unit tests for the step closure conversion and the updated evaluator
  function.

## Capabilities

### New Capabilities

- `step-closure-conversion`: Convert provider Step data structs into gemara
  AssessmentStep closures so evaluation logs contain per-step assessment data.

### Modified Capabilities

## Impact

- **Code**: `internal/output/evaluator.go` (primary), `internal/output/sarif.go`
  (potential reuse of `providerResultToGemara`).
- **Output**: Evaluation log YAML will now contain populated `steps` arrays,
  changing the output shape for all scan formats that consume `gemara.EvaluationLog`.
- **Downstream**: SARIF converter in `gemaraconv` will gain access to step data
  for `LogicalLocation` enrichment. Markdown and OSCAL formatters are unaffected
  (they already use `StepsExecuted`).
- **Dependencies**: No new dependencies. Uses existing `go-gemara` types.
- **API**: No proto or gRPC changes required. The provider-side data is already
  transmitted correctly.
