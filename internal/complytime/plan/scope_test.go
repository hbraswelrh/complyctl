// SPDX-License-Identifier: Apache-2.0
package plan

import (
	"testing"

	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

// FIXME: Expand test cases
func TestAssessmentScope_ApplyScope(t *testing.T) {
	testLogger := hclog.NewNullLogger()

	tests := []struct {
		name           string
		basePlan       *oscalTypes.AssessmentPlan
		scope          AssessmentScope
		wantSelections []oscalTypes.AssessedControls
	}{
		{
			name: "Success/Default",
			basePlan: &oscalTypes.AssessmentPlan{
				ReviewedControls: oscalTypes.ReviewedControls{
					ControlSelections: []oscalTypes.AssessedControls{
						{
							IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
								{
									ControlId: "example-1",
								},
								{
									ControlId: "example-2",
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID:     "test",
				IncludeControls: []string{"example-2"},
			},
			wantSelections: []oscalTypes.AssessedControls{
				{
					IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
						{
							ControlId: "example-2",
						},
					},
				},
			},
		},
		// Testing for out-of-scope controls
		{
			name: "All Controls Out-of-Scope",
			basePlan: &oscalTypes.AssessmentPlan{
				ReviewedControls: oscalTypes.ReviewedControls{
					ControlSelections: []oscalTypes.AssessedControls{
						{
							IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
								{
									ControlId: "",
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID:     "test",
				IncludeControls: nil,
			},
			wantSelections: []oscalTypes.AssessedControls{
				{
					IncludeControls: nil,
				},
			},
		},
		{
			name: "Some Controls Out-of-Scope",
			basePlan: &oscalTypes.AssessmentPlan{
				ReviewedControls: oscalTypes.ReviewedControls{
					ControlSelections: []oscalTypes.AssessedControls{
						{
							IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
								{
									ControlId: "example-1",
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID:     "test",
				IncludeControls: []string{"example-1", "example-2"},
			},
			wantSelections: []oscalTypes.AssessedControls{
				{
					IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
						{
							ControlId: "example-1",
						},
					},
				},
			},
		},
	}
	{
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := tt.scope
			scope.ApplyScope(tt.basePlan, testLogger)
			require.Equal(t, tt.wantSelections, tt.basePlan.ReviewedControls.ControlSelections)
		})
	}
}
