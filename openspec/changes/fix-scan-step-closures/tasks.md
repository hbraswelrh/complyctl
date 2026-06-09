## 1. Step Closure Conversion Function

- [x] 1.1 Add `providerStepToGemara(step provider.Step, confidence provider.ConfidenceLevel) gemara.AssessmentStep` function in `internal/output/evaluator.go` that creates a closure returning `(providerResultToGemara(step.Result), step.Message, providerConfidenceToGemara(confidence))`
- [x] 1.2 Add `providerStepsToGemara(steps []provider.Step, confidence provider.ConfidenceLevel) []gemara.AssessmentStep` helper that maps a slice of provider steps to gemara step closures

## 2. Evaluator Integration

- [x] 2.1 Update `providerToGemaraAssessment()` in `internal/output/evaluator.go` to populate the `Steps` field on the returned `gemara.AssessmentLog` using `providerStepsToGemara(a.Steps, a.Confidence)`

## 3. Unit Tests

- [x] 3.1 Add test `TestProviderStepToGemara_SingleStep` verifying a single closure returns the correct result, message, and confidence
- [x] 3.2 Add test `TestProviderStepsToGemara_MultipleSteps` verifying slice conversion preserves order and count
- [x] 3.3 Add test `TestProviderStepsToGemara_EmptySteps` verifying empty input returns nil/empty slice
- [x] 3.4 Add test `TestProviderStepToGemara_ResultMapping` verifying each provider.Result maps to the correct gemara.Result (Passed, Failed, Skipped, Error, unrecognized)
- [x] 3.5 Add test `TestProviderToGemaraAssessment_StepsPopulated` verifying the full `providerToGemaraAssessment()` method now populates `Steps` with the correct count and callable closures

## 4. Verification

- [x] 4.1 Run `make test-unit` and confirm all tests pass
- [x] 4.2 Run `make lint` and confirm no lint errors
- [x] 4.3 Run `make vet` and confirm no vet issues
