## ADDED Requirements

### Requirement: Provider steps converted to gemara AssessmentStep closures

The evaluator SHALL convert each `provider.Step` in a `provider.AssessmentLog`
into a `gemara.AssessmentStep` closure when building the `gemara.AssessmentLog`.
Each closure SHALL return the step's pre-computed Result (mapped to
`gemara.Result`), Message, and the parent assessment's ConfidenceLevel (mapped
to `gemara.ConfidenceLevel`).

#### Scenario: Single step produces a populated closure

- **WHEN** a provider returns an `AssessmentLog` with one `Step` having
  Name="check-permissions", Result=Passed, Message="all permissions valid"
- **THEN** the resulting `gemara.AssessmentLog.Steps` SHALL contain exactly
  one `AssessmentStep` closure that, when invoked with any payload, returns
  `(gemara.Passed, "all permissions valid", <mapped confidence>)`

#### Scenario: Multiple steps produce ordered closures

- **WHEN** a provider returns an `AssessmentLog` with three `Step` entries
- **THEN** the resulting `gemara.AssessmentLog.Steps` SHALL contain three
  `AssessmentStep` closures in the same order as the provider steps

#### Scenario: Zero steps produce empty Steps slice

- **WHEN** a provider returns an `AssessmentLog` with zero `Step` entries
- **THEN** the resulting `gemara.AssessmentLog.Steps` SHALL be nil or empty
  and `StepsExecuted` SHALL be 0

### Requirement: Result mapping preserves provider semantics

The conversion from `provider.Result` to `gemara.Result` within step closures
SHALL use the same mapping as the existing `providerResultToGemara` function:
Passed to Passed, Failed to Failed, Skipped to NotApplicable, Error to Unknown,
and unrecognized values to NotRun.

#### Scenario: Each provider result maps correctly

- **WHEN** a provider step has Result=Failed
- **THEN** the closure SHALL return `gemara.Failed`

#### Scenario: Unrecognized result maps to NotRun

- **WHEN** a provider step has an unrecognized Result value
- **THEN** the closure SHALL return `gemara.NotRun`

### Requirement: Confidence mapping uses assessment-level confidence

Each step closure SHALL use the parent `provider.AssessmentLog.Confidence`
value (mapped to `gemara.ConfidenceLevel`) since provider steps do not carry
per-step confidence.

#### Scenario: Assessment confidence propagates to all step closures

- **WHEN** an `AssessmentLog` has Confidence=High and contains two steps
- **THEN** both step closures SHALL return `gemara.High` as their
  ConfidenceLevel

### Requirement: Evaluation log YAML contains populated steps

The YAML serialization of the `gemara.EvaluationLog` produced by the evaluator
SHALL contain non-empty `steps` arrays in each `AssessmentLog` that has provider
steps, enabling Gemara schema validation to pass.

#### Scenario: YAML output contains step entries

- **WHEN** a scan produces assessment logs with provider steps
- **THEN** the written evaluation log YAML SHALL contain `steps:` arrays with
  one entry per provider step, serialized via `AssessmentStep.MarshalYAML()`
