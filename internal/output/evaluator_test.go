// SPDX-License-Identifier: Apache-2.0

package output_test

import (
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/complytime/complyctl/internal/output"
	"github.com/complytime/complyctl/pkg/provider"
)

func TestGemaraLog_MetadataType(t *testing.T) {
	eval := output.NewEvaluator("test-policy", "target-1", nil)
	eval.AddTarget([]provider.AssessmentLog{
		{
			RequirementID: "REQ-1",
			Steps:         []provider.Step{{Result: provider.ResultPassed, Message: "ok"}},
		},
	})

	log := eval.GemaraLog()
	assert.Equal(t, gemara.EvaluationLogArtifact, log.Metadata.Type)
	assert.Equal(t, gemara.SchemaVersion, log.Metadata.GemaraVersion)
}

func TestGemaraLog_AggregatesResult(t *testing.T) {
	tests := []struct {
		name     string
		steps    []provider.Step
		expected gemara.Result
	}{
		{
			name:     "all passing yields Passed",
			steps:    []provider.Step{{Result: provider.ResultPassed, Message: "ok"}},
			expected: gemara.Passed,
		},
		{
			name:     "any failure yields Failed",
			steps:    []provider.Step{{Result: provider.ResultFailed, Message: "bad"}},
			expected: gemara.Failed,
		},
		{
			name:     "no steps yields NotRun",
			steps:    nil,
			expected: gemara.NotRun,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := output.NewEvaluator("pol", "tgt", nil)
			eval.AddTarget([]provider.AssessmentLog{
				{RequirementID: "R1", Steps: tt.steps},
			})
			assert.Equal(t, tt.expected, eval.GemaraLog().Result)
		})
	}
}

func TestGemaraLog_PopulatesTarget(t *testing.T) {
	eval := output.NewEvaluator("policy-id", "my-target", nil)
	eval.AddTarget([]provider.AssessmentLog{
		{RequirementID: "R1", Steps: []provider.Step{{Result: provider.ResultPassed, Message: "ok"}}},
	})

	log := eval.GemaraLog()
	assert.Equal(t, "my-target", log.Target.Id)
	assert.Equal(t, "my-target", log.Target.Name)
	assert.Equal(t, gemara.Software, log.Target.Type)
}

func TestGemaraLog_AssessmentMessageUsesStepViolation(t *testing.T) {
	eval := output.NewEvaluator("pol", "tgt", nil)
	eval.AddTarget([]provider.AssessmentLog{
		{
			RequirementID: "REQ-1",
			Steps: []provider.Step{
				{Result: provider.ResultFailed, Message: "cert validity exceeds 397 days"},
			},
			Message: "0 of 1 targets passed",
		},
	})

	log := eval.GemaraLog()
	require.Len(t, log.Evaluations, 1)
	require.Len(t, log.Evaluations[0].AssessmentLogs, 1)
	assert.Equal(t, "cert validity exceeds 397 days", log.Evaluations[0].AssessmentLogs[0].Message)
}

func TestGemaraLog_PassingAssessmentKeepsProviderMessage(t *testing.T) {
	eval := output.NewEvaluator("pol", "tgt", nil)
	eval.AddTarget([]provider.AssessmentLog{
		{
			RequirementID: "REQ-1",
			Steps: []provider.Step{
				{Result: provider.ResultPassed, Message: "check ok"},
			},
			Message: "1 of 1 targets passed",
		},
	})

	log := eval.GemaraLog()
	require.Len(t, log.Evaluations, 1)
	require.Len(t, log.Evaluations[0].AssessmentLogs, 1)
	assert.Equal(t, "1 of 1 targets passed", log.Evaluations[0].AssessmentLogs[0].Message)
}

func TestEvaluator_Write(t *testing.T) {
	outDir := t.TempDir()
	eval := output.NewEvaluator("test-policy", "target-1", nil)
	eval.AddTarget([]provider.AssessmentLog{
		{RequirementID: "R1", Steps: []provider.Step{{Result: provider.ResultPassed, Message: "ok"}}},
	})

	path, err := eval.Write(outDir)
	require.NoError(t, err)
	assert.FileExists(t, path)
	assert.Contains(t, path, "evaluation-log-test-policy-target-1-")
}
