// SPDX-License-Identifier: Apache-2.0

package output

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/complytime/complyctl/pkg/provider"
)

func TestFormatScanSummary_SingleTarget(t *testing.T) {
	assessments := []provider.AssessmentLog{
		{
			RequirementID: "REQ-1",
			Steps: []provider.Step{
				{Result: provider.ResultPassed, Message: "Test passed"},
			},
		},
		{
			RequirementID: "REQ-2",
			Steps: []provider.Step{
				{Result: provider.ResultFailed, Message: "Test failed"},
			},
		},
	}
	assessmentTargets := []string{"target1", "target1"}
	reqToControl := map[string]string{
		"REQ-1": "CTRL-1",
		"REQ-2": "CTRL-2",
	}
	policyID := "test-policy"
	targetIDs := []string{"target1"}

	output := FormatScanSummary(assessments, assessmentTargets, reqToControl, policyID, targetIDs)

	assert.Contains(t, output, "Scan: test-policy")
	assert.Contains(t, output, "Target: target1")
	assert.Contains(t, output, "2 requirements")
	assert.Contains(t, output, "TARGET ID")
	assert.Contains(t, output, "REQUIREMENT ID")
	assert.Contains(t, output, "target1")
	assert.Contains(t, output, "REQ-2")
	assert.Contains(t, output, "1 passed, 1 failed")
}

func TestFormatScanSummary_MultipleTargets(t *testing.T) {
	assessments := []provider.AssessmentLog{
		{
			RequirementID: "REQ-1",
			Steps: []provider.Step{
				{Result: provider.ResultFailed, Message: "Target A failed"},
			},
		},
		{
			RequirementID: "REQ-1",
			Steps: []provider.Step{
				{Result: provider.ResultFailed, Message: "Target B failed"},
			},
		},
	}
	assessmentTargets := []string{"targetA", "targetB"}
	reqToControl := map[string]string{"REQ-1": "CTRL-1"}
	policyID := "multi-target-policy"
	targetIDs := []string{"targetA", "targetB"}

	output := FormatScanSummary(assessments, assessmentTargets, reqToControl, policyID, targetIDs)

	assert.Contains(t, output, "Target: targetA, targetB")
	assert.Contains(t, output, "2 requirements")
	lines := strings.Split(output, "\n")
	var targetAFound, targetBFound bool
	for _, line := range lines {
		if strings.Contains(line, "targetA") && strings.Contains(line, "REQ-1") {
			targetAFound = true
		}
		if strings.Contains(line, "targetB") && strings.Contains(line, "REQ-1") {
			targetBFound = true
		}
	}
	assert.True(t, targetAFound, "targetA should appear in output")
	assert.True(t, targetBFound, "targetB should appear in output")
}

func TestFormatScanSummary_AllPassed(t *testing.T) {
	assessments := []provider.AssessmentLog{
		{
			RequirementID: "REQ-1",
			Steps:         []provider.Step{{Result: provider.ResultPassed, Message: "OK"}},
		},
		{
			RequirementID: "REQ-2",
			Steps:         []provider.Step{{Result: provider.ResultPassed, Message: "OK"}},
		},
	}
	assessmentTargets := []string{"target1", "target1"}
	reqToControl := map[string]string{}
	policyID := "passing-policy"
	targetIDs := []string{"target1"}

	output := FormatScanSummary(assessments, assessmentTargets, reqToControl, policyID, targetIDs)

	assert.Contains(t, output, "2 requirements: 2 passed, 0 failed, 0 not applicable, 0 skipped, 0 errors")
	assert.NotContains(t, output, "TARGET ID")
}

func TestFormatScanSummary_MissingTargetID(t *testing.T) {
	assessments := []provider.AssessmentLog{
		{
			RequirementID: "REQ-1",
			Steps:         []provider.Step{{Result: provider.ResultFailed, Message: "Failed"}},
		},
	}
	assessmentTargets := []string{}
	reqToControl := map[string]string{"REQ-1": "CTRL-1"}
	policyID := "test"
	targetIDs := []string{"target1"}

	output := FormatScanSummary(assessments, assessmentTargets, reqToControl, policyID, targetIDs)

	require.Contains(t, output, "1 requirements")
	assert.Contains(t, output, "-")
}

func TestFormatOperationalWarnings_Empty(t *testing.T) {
	result := FormatOperationalWarnings(nil)
	assert.Empty(t, result)

	result = FormatOperationalWarnings([]string{})
	assert.Empty(t, result)
}

func TestFormatOperationalWarnings_SingleError(t *testing.T) {
	result := FormatOperationalWarnings([]string{"target 'staging': clone failed: auth denied"})

	assert.Contains(t, result, "WARNING: 1 operational error during scan")
	assert.Contains(t, result, "clone failed: auth denied")
}

func TestFormatOperationalWarnings_MultipleErrors(t *testing.T) {
	errors := []string{
		"target 'staging': clone failed: auth denied",
		"target 'dev': missing required tool: conftest",
	}
	result := FormatOperationalWarnings(errors)

	assert.Contains(t, result, "WARNING: 2 operational errors during scan")
	assert.Contains(t, result, "  - target 'staging': clone failed: auth denied")
	assert.Contains(t, result, "  - target 'dev': missing required tool: conftest")
}

func TestFormatScanSummary_NotRunCountedAsSkipped(t *testing.T) {
	assessments := []provider.AssessmentLog{
		{
			RequirementID: "REQ-1",
			Steps:         nil, // no steps → NotRun → counted as skipped
		},
		{
			RequirementID: "REQ-2",
			Steps:         []provider.Step{{Result: provider.ResultPassed, Message: "OK"}},
		},
	}
	assessmentTargets := []string{"target1", "target1"}
	reqToControl := map[string]string{}
	policyID := "test"
	targetIDs := []string{"target1"}

	result := FormatScanSummary(assessments, assessmentTargets, reqToControl, policyID, targetIDs)

	assert.Contains(t, result, "1 skipped")
	assert.Contains(t, result, "1 passed")
	assert.Contains(t, result, "0 not applicable")
}

func TestFormatScanSummary_ZeroAssessments(t *testing.T) {
	result := FormatScanSummary(nil, nil, map[string]string{}, "test", []string{"target1"})

	assert.Contains(t, result, "0 requirements: 0 passed, 0 failed, 0 not applicable, 0 skipped, 0 errors")
}

func TestNothingAssessed_EmptyAssessments(t *testing.T) {
	assert.True(t, NothingAssessed(nil))
	assert.True(t, NothingAssessed([]provider.AssessmentLog{}))
}

func TestNothingAssessed_AllNotRun(t *testing.T) {
	assessments := []provider.AssessmentLog{
		{
			RequirementID: "REQ-1",
			Steps:         nil, // no steps → NotRun → nothing assessed
		},
	}
	assert.True(t, NothingAssessed(assessments))
}

func TestNothingAssessed_WithPassedResult(t *testing.T) {
	assessments := []provider.AssessmentLog{
		{
			RequirementID: "REQ-1",
			Steps:         []provider.Step{{Result: provider.ResultPassed, Message: "OK"}},
		},
	}
	assert.False(t, NothingAssessed(assessments))
}

func TestNothingAssessed_WithFailedResult(t *testing.T) {
	assessments := []provider.AssessmentLog{
		{
			RequirementID: "REQ-1",
			Steps:         []provider.Step{{Result: provider.ResultFailed, Message: "Fail"}},
		},
	}
	assert.False(t, NothingAssessed(assessments))
}

func TestFormatScanSummary_ControlIDMissing(t *testing.T) {
	assessments := []provider.AssessmentLog{
		{
			RequirementID: "REQ-UNKNOWN",
			Steps:         []provider.Step{{Result: provider.ResultFailed, Message: "No control"}},
		},
	}
	assessmentTargets := []string{"target1"}
	reqToControl := map[string]string{}
	policyID := "test"
	targetIDs := []string{"target1"}

	output := FormatScanSummary(assessments, assessmentTargets, reqToControl, policyID, targetIDs)

	lines := strings.Split(output, "\n")
	var foundDash bool
	for _, line := range lines {
		if strings.Contains(line, "REQ-UNKNOWN") && strings.Contains(line, "target1") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "REQ-UNKNOWN" && i+1 < len(parts) && parts[i+1] == "-" {
					foundDash = true
					break
				}
			}
		}
	}
	assert.True(t, foundDash, "Should show '-' for missing control ID")
}
