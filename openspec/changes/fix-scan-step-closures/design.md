## Context

The `complyctl scan` command collects `provider.AssessmentLog` results from
provider plugins via gRPC and converts them into `gemara.EvaluationLog` for
output. The conversion lives in `internal/output/evaluator.go`, specifically
the `providerToGemaraAssessment()` method (line 121).

The provider pipeline works correctly end-to-end: providers populate
`provider.Step` structs with Name, Result, and Message; the gRPC layer
serializes/deserializes them faithfully via `api/plugin/plugin.proto` Step
messages; and the evaluator receives intact `[]provider.Step` slices.

The problem is at the final conversion step. `gemara.AssessmentLog.Steps` is
typed as `[]gemara.AssessmentStep`, where `AssessmentStep` is a function type
`func(payload interface{}) (Result, string, ConfidenceLevel)`. The evaluator
never populates this field, so the YAML output always contains `steps: []`.

Two helper functions already exist in `internal/output/sarif.go`:
- `providerResultToGemara()` (line 42): maps `provider.Result` to `gemara.Result`
- `aggregateResultFromSteps()` (line 57): iterates provider steps and aggregates

These are package-private and already used by the evaluator.

## Goals / Non-Goals

**Goals:**
- Populate `gemara.AssessmentLog.Steps` with closures that return each
  provider step's result, message, and confidence level.
- Produce evaluation log YAML that passes Gemara schema validation (non-empty
  `steps` arrays).
- Maintain backward compatibility with all existing output formats (SARIF,
  OSCAL, Markdown).

**Non-Goals:**
- Making the closures re-executable against live data. These are data-carrying
  closures that replay fixed results; they are not live assessment functions.
- Changing the provider gRPC API or proto definitions.
- Modifying the gemara library itself.
- Adding step-level detail to Markdown or OSCAL output formats (future work).

## Decisions

### D1: Wrap provider Steps as data-carrying closures

Each `provider.Step` is converted to a `gemara.AssessmentStep` closure that
ignores its `payload` argument and returns the step's pre-computed result,
message, and confidence level:

```go
func providerStepToGemara(step provider.Step, confidence provider.ConfidenceLevel) gemara.AssessmentStep {
    result := providerResultToGemara(step.Result)
    conf := providerConfidenceToGemara(confidence)
    return func(_ interface{}) (gemara.Result, string, gemara.ConfidenceLevel) {
        return result, step.Message, conf
    }
}
```

**Rationale**: The gemara `AssessmentStep` type is designed for executable
steps in a live evaluation engine. In the complyctl context, the evaluation
has already been performed by the provider. The closure pattern satisfies
the type contract while preserving the step data for serialization. The
`MarshalYAML()` and `MarshalJSON()` methods on `AssessmentStep` use
`runtime.FuncForPC()` to serialize the function name, which will produce a
stable identifier for these closures.

**Alternative considered**: Creating a separate struct that implements a
hypothetical `AssessmentStep` interface. Rejected because `AssessmentStep`
is a function type, not an interface, so no struct-based alternative is
possible without modifying the gemara library.

### D2: Use assessment-level confidence for all steps

The `provider.Step` struct does not carry a per-step confidence level. The
closure uses the parent `provider.AssessmentLog.Confidence` value for all
steps within that assessment.

**Rationale**: The provider gRPC API defines confidence at the assessment
level, not the step level. Using the assessment-level confidence is the
most accurate representation of what the provider communicated.

### D3: Place the conversion function in evaluator.go

The new `providerStepToGemara()` function lives in `evaluator.go` alongside
`providerToGemaraAssessment()` and `providerConfidenceToGemara()`, since it
is part of the same provider-to-gemara conversion pipeline.

**Rationale**: Keeps the conversion logic colocated. The existing
`providerResultToGemara()` in `sarif.go` is already called from
`aggregateResultFromSteps()` which is used by the evaluator. No relocation
of existing functions is needed.

## Risks / Trade-offs

- **[YAML serialization of closures]** The `AssessmentStep.MarshalYAML()`
  method uses `runtime.FuncForPC()` to extract the function name. For
  anonymous closures, this produces names like
  `github.com/complytime/complyctl/internal/output.providerStepToGemara.func1`.
  This is an implementation detail and not a human-readable step name.
  Mitigation: This matches how go-gemara itself serializes steps. The step
  Name field from the provider is preserved in the closure's message return
  value. Future gemara versions may add a `Name` field to improve
  serialization.

- **[Test stability]** Function pointer names from `runtime.FuncForPC()` may
  vary across Go versions or build configurations. Mitigation: Unit tests
  should verify closure behavior (return values) rather than serialized
  function names.
