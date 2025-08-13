// SPDX-License-Identifier: Apache-2.0
package complytime

import (
	"testing"

	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/hashicorp/go-hclog"
	"github.com/oscal-compass/oscal-sdk-go/extensions"
	"github.com/oscal-compass/oscal-sdk-go/validation"
	"github.com/stretchr/testify/require"
)

func TestNewAssessmentScopeFromCDs(t *testing.T) {
	testAppDir := ApplicationDirectory{}
	validator := validation.NoopValidator{}

	_, err := NewAssessmentScopeFromCDs("example", testAppDir, validator)
	require.EqualError(t, err, "no component definitions found")

	cd := oscalTypes.ComponentDefinition{
		Components: &[]oscalTypes.DefinedComponent{
			{
				Title: "Component",
				ControlImplementations: &[]oscalTypes.ControlImplementationSet{
					{
						Props: &[]oscalTypes.Property{
							{
								Name:  extensions.FrameworkProp,
								Value: "example",
								Ns:    extensions.TrestleNameSpace,
							},
						},
						SetParameters: &[]oscalTypes.SetParameter{
							{
								ParamId: "param-1",
								Values:  []string{"value-1", "value-2"},
							},
							{
								ParamId: "param-2",
								Values:  []string{"value-3"},
							},
						},
						ImplementedRequirements: []oscalTypes.ImplementedRequirementControlImplementation{
							{
								ControlId: "control-1",
							},
							{
								ControlId: "control-2",
							},
						},
					},
				},
			},
		},
	}

	wantScope := AssessmentScope{
		FrameworkID: "example",
		IncludeControls: []ControlEntry{
			{
				ControlID:    "control-1",
				ControlTitle: "",
				IncludeRules: []string{"*"},
				SelectParameters: []ParameterEntry{
					{Name: "param-1", Value: "value-1"},
					{Name: "param-2", Value: "value-3"},
				},
			},
			{
				ControlID:    "control-2",
				ControlTitle: "",
				IncludeRules: []string{"*"},
				SelectParameters: []ParameterEntry{
					{Name: "param-1", Value: "value-1"},
					{Name: "param-2", Value: "value-3"},
				},
			},
		},
	}
	scope, err := NewAssessmentScopeFromCDs("example", testAppDir, validator, cd)
	require.NoError(t, err)

	// Check the basic structure
	require.Equal(t, wantScope.FrameworkID, scope.FrameworkID)
	require.Len(t, scope.IncludeControls, len(wantScope.IncludeControls))

	// Check each control entry, allowing for different parameter orders
	for i, wantControl := range wantScope.IncludeControls {
		actualControl := scope.IncludeControls[i]
		require.Equal(t, wantControl.ControlID, actualControl.ControlID)
		require.Equal(t, wantControl.ControlTitle, actualControl.ControlTitle)
		require.Equal(t, wantControl.IncludeRules, actualControl.IncludeRules)

		// Check parameters exist regardless of order
		require.Len(t, actualControl.SelectParameters, len(wantControl.SelectParameters))
		for _, wantParam := range wantControl.SelectParameters {
			found := false
			for _, actualParam := range actualControl.SelectParameters {
				if actualParam.Name == wantParam.Name && actualParam.Value == wantParam.Value {
					found = true
					break
				}
			}
			require.True(t, found, "Expected parameter %s=%s not found", wantParam.Name, wantParam.Value)
		}
	}

	// Reproduce duplicates
	anotherComponent := oscalTypes.DefinedComponent{
		Title: "AnotherComponent",
		ControlImplementations: &[]oscalTypes.ControlImplementationSet{
			{
				Props: &[]oscalTypes.Property{
					{
						Name:  extensions.FrameworkProp,
						Value: "example",
						Ns:    extensions.TrestleNameSpace,
					},
				},
				ImplementedRequirements: []oscalTypes.ImplementedRequirementControlImplementation{
					{
						ControlId: "control-1",
					},
					{
						ControlId: "control-2",
					},
				},
			},
		},
	}
	*cd.Components = append(*cd.Components, anotherComponent)

	scope, err = NewAssessmentScopeFromCDs("example", testAppDir, validator, cd)
	require.NoError(t, err)

	// Check the basic structure again after adding duplicates
	require.Equal(t, wantScope.FrameworkID, scope.FrameworkID)
	require.Len(t, scope.IncludeControls, len(wantScope.IncludeControls))

	// Check each control entry again, allowing for different parameter orders
	for i, wantControl := range wantScope.IncludeControls {
		actualControl := scope.IncludeControls[i]
		require.Equal(t, wantControl.ControlID, actualControl.ControlID)
		require.Equal(t, wantControl.ControlTitle, actualControl.ControlTitle)
		require.Equal(t, wantControl.IncludeRules, actualControl.IncludeRules)

		// Check parameters exist regardless of order
		require.Len(t, actualControl.SelectParameters, len(wantControl.SelectParameters))
		for _, wantParam := range wantControl.SelectParameters {
			found := false
			for _, actualParam := range actualControl.SelectParameters {
				if actualParam.Name == wantParam.Name && actualParam.Value == wantParam.Value {
					found = true
					break
				}
			}
			require.True(t, found, "Expected parameter %s=%s not found", wantParam.Name, wantParam.Value)
		}
	}
}

func TestNewAssessmentScopeFromCDs_NoParameters(t *testing.T) {
	testAppDir := ApplicationDirectory{}
	validator := validation.NoopValidator{}

	cd := oscalTypes.ComponentDefinition{
		Components: &[]oscalTypes.DefinedComponent{
			{
				Title: "Component",
				ControlImplementations: &[]oscalTypes.ControlImplementationSet{
					{
						Props: &[]oscalTypes.Property{
							{
								Name:  extensions.FrameworkProp,
								Value: "example",
								Ns:    extensions.TrestleNameSpace,
							},
						},
						ImplementedRequirements: []oscalTypes.ImplementedRequirementControlImplementation{
							{
								ControlId: "control-1",
								// No SetParameters
							},
						},
					},
				},
			},
		},
	}

	wantScope := AssessmentScope{
		FrameworkID: "example",
		IncludeControls: []ControlEntry{
			{
				ControlID:    "control-1",
				ControlTitle: "",
				IncludeRules: []string{"*"},
				SelectParameters: []ParameterEntry{
					{Name: "N/A", Value: "N/A"},
				},
			},
		},
	}

	scope, err := NewAssessmentScopeFromCDs("example", testAppDir, validator, cd)
	require.NoError(t, err)
	require.Equal(t, wantScope, scope)
}

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
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{ControlID: "example-2", IncludeRules: []string{"*"}},
				},
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
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{ControlID: "example-1", IncludeRules: []string{"*"}},
					{ControlID: "example-2", IncludeRules: []string{"*"}},
				},
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := tt.scope
			err := scope.ApplyScope(tt.basePlan, testLogger)
			require.NoError(t, err)
			require.Equal(t, tt.wantSelections, tt.basePlan.ReviewedControls.ControlSelections)
		})
	}
}

func TestAssessmentScope_ApplyParameterScope(t *testing.T) {
	testLogger := hclog.NewNullLogger()

	tests := []struct {
		name           string
		basePlan       *oscalTypes.AssessmentPlan
		scope          AssessmentScope
		wantActivities *[]oscalTypes.Activity
	}{
		{
			name: "Success/ParameterUpdate",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "test-activity",
							Props: &[]oscalTypes.Property{
								{
									Name:  "test-param",
									Value: "default-value",
									Class: extensions.TestParameterClass,
								},
								{
									Name:  "other-param",
									Value: "other-value",
									Class: "other-class",
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{
						ControlID: "control-1",
						SelectParameters: []ParameterEntry{
							{Name: "test-param", Value: "custom-value"},
						},
					},
				},
			},
			wantActivities: &[]oscalTypes.Activity{
				{
					Title: "test-activity",
					Props: &[]oscalTypes.Property{
						{
							Name:  "test-param",
							Value: "custom-value",
							Class: extensions.TestParameterClass,
						},
						{
							Name:  "other-param",
							Value: "other-value",
							Class: "other-class",
						},
					},
				},
			},
		},
		{
			name: "Success/NoParametersToUpdate",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "test-activity",
							Props: &[]oscalTypes.Property{
								{
									Name:  "some-param",
									Value: "default-value",
									Class: "other-class",
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{
						ControlID: "control-1",
						SelectParameters: []ParameterEntry{
							{Name: "different-param", Value: "custom-value"},
						},
					},
				},
			},
			wantActivities: &[]oscalTypes.Activity{
				{
					Title: "test-activity",
					Props: &[]oscalTypes.Property{
						{
							Name:  "some-param",
							Value: "default-value",
							Class: "other-class",
						},
					},
				},
			},
		},
		{
			name: "Success/EmptyParameterName",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "test-activity",
							Props: &[]oscalTypes.Property{
								{
									Name:  "test-param",
									Value: "default-value",
									Class: extensions.TestParameterClass,
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{
						ControlID: "control-1",
						SelectParameters: []ParameterEntry{
							{Name: "", Value: "should-be-ignored"},
							{Name: "test-param", Value: "custom-value"},
						},
					},
				},
			},
			wantActivities: &[]oscalTypes.Activity{
				{
					Title: "test-activity",
					Props: &[]oscalTypes.Property{
						{
							Name:  "test-param",
							Value: "custom-value",
							Class: extensions.TestParameterClass,
						},
					},
				},
			},
		},
		{
			name: "Success/NoSelectParameters",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "test-activity",
							Props: &[]oscalTypes.Property{
								{
									Name:  "test-param",
									Value: "default-value",
									Class: extensions.TestParameterClass,
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{
						ControlID: "control-1",
					},
				},
			},
			wantActivities: &[]oscalTypes.Activity{
				{
					Title: "test-activity",
					Props: &[]oscalTypes.Property{
						{
							Name:  "test-param",
							Value: "default-value",
							Class: extensions.TestParameterClass,
						},
					},
				},
			},
		},
		{
			name:     "Success/NoLocalDefinitions",
			basePlan: &oscalTypes.AssessmentPlan{},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{
						ControlID: "control-1",
						SelectParameters: []ParameterEntry{
							{Name: "test-param", Value: "custom-value"},
						},
					},
				},
			},
			wantActivities: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := tt.scope
			err := scope.ApplyScope(tt.basePlan, testLogger)
			require.NoError(t, err)
			if tt.basePlan.LocalDefinitions != nil {
				require.Equal(t, tt.wantActivities, tt.basePlan.LocalDefinitions.Activities)
			} else {
				require.Nil(t, tt.wantActivities)
			}
		})
	}
}

func TestAssessmentScope_ParameterValidation(t *testing.T) {
	testLogger := hclog.NewNullLogger()

	tests := []struct {
		name          string
		basePlan      *oscalTypes.AssessmentPlan
		scope         AssessmentScope
		expectError   bool
		errorContains string
	}{
		{
			name: "Error/InvalidParameterValue",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "test-activity",
							Props: &[]oscalTypes.Property{
								{
									Name:    extensions.ParameterIdProp,
									Value:   "test-param",
									Remarks: "test-remarks",
								},
								{
									Name:    "test-param",
									Value:   "default-value",
									Class:   extensions.TestParameterClass,
									Remarks: "test-remarks",
								},
								{
									Name:    "Parameter_Value_Alternatives_1",
									Value:   `{"option1": "valid-value-1", "option2": "valid-value-2"}`,
									Remarks: "test-remarks",
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{
						ControlID: "control-1",
						SelectParameters: []ParameterEntry{
							{Name: "test-param", Value: "invalid-value"},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "parameter 'test-param' has invalid value 'invalid-value'",
		},
		{
			name: "Success/ValidParameterValue",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "test-activity",
							Props: &[]oscalTypes.Property{
								{
									Name:    extensions.ParameterIdProp,
									Value:   "test-param",
									Remarks: "test-remarks",
								},
								{
									Name:    "test-param",
									Value:   "default-value",
									Class:   extensions.TestParameterClass,
									Remarks: "test-remarks",
								},
								{
									Name:    "Parameter_Value_Alternatives_1",
									Value:   `{"option1": "valid-value-1", "option2": "valid-value-2"}`,
									Remarks: "test-remarks",
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{
						ControlID: "control-1",
						SelectParameters: []ParameterEntry{
							{Name: "test-param", Value: "valid-value-1"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Success/NoAlternativesDefinedAnyValueAccepted",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "test-activity",
							Props: &[]oscalTypes.Property{
								{
									Name:  "test-param",
									Value: "default-value",
									Class: extensions.TestParameterClass,
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{
						ControlID: "control-1",
						SelectParameters: []ParameterEntry{
							{Name: "test-param", Value: "any-value"},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := tt.scope
			err := scope.ApplyScope(tt.basePlan, testLogger)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAssessmentScope_ApplyRuleScope(t *testing.T) {
	testLogger := hclog.NewNullLogger()

	tests := []struct {
		name           string
		basePlan       *oscalTypes.AssessmentPlan
		scope          AssessmentScope
		wantActivities *[]oscalTypes.Activity
	}{
		{
			name: "Success/Default",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "rule-1",
							RelatedControls: &oscalTypes.ReviewedControls{
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
						{
							Title: "rule-2",
							RelatedControls: &oscalTypes.ReviewedControls{
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
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{ControlID: "example-1", IncludeRules: []string{"*"}},
					{ControlID: "example-2", IncludeRules: []string{"*"}},
				},
			},
			wantActivities: &[]oscalTypes.Activity{
				{
					Title: "rule-1",
					RelatedControls: &oscalTypes.ReviewedControls{
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
				{
					Title: "rule-2",
					RelatedControls: &oscalTypes.ReviewedControls{
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
			},
		},
		{
			name: "Success/ExcludeRuleForControl",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "rule-1",
							RelatedControls: &oscalTypes.ReviewedControls{
								ControlSelections: []oscalTypes.AssessedControls{
									{
										IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
											{
												ControlId: "control-1",
											},
											{
												ControlId: "control-2",
											},
										},
									},
								},
							},
						},
						{
							Title: "rule-2",
							RelatedControls: &oscalTypes.ReviewedControls{
								ControlSelections: []oscalTypes.AssessedControls{
									{
										IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
											{
												ControlId: "control-1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{ControlID: "control-1", IncludeRules: []string{"*"}, ExcludeRules: []string{"rule-1"}},
					{ControlID: "control-2", IncludeRules: []string{"*"}},
				},
			},
			wantActivities: &[]oscalTypes.Activity{
				{
					Title: "rule-1",
					RelatedControls: &oscalTypes.ReviewedControls{
						ControlSelections: []oscalTypes.AssessedControls{
							{
								IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
									{
										ControlId: "control-2",
									},
								},
							},
						},
					},
				},
				{
					Title: "rule-2",
					RelatedControls: &oscalTypes.ReviewedControls{
						ControlSelections: []oscalTypes.AssessedControls{
							{
								IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
									{
										ControlId: "control-1",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Success/ActivityMarkedSkipped",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "rule-1",
							RelatedControls: &oscalTypes.ReviewedControls{
								ControlSelections: []oscalTypes.AssessedControls{
									{
										IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
											{
												ControlId: "control-1",
											},
										},
									},
								},
							},
						},
						{
							Title: "rule-2",
							RelatedControls: &oscalTypes.ReviewedControls{
								ControlSelections: []oscalTypes.AssessedControls{
									{
										IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
											{
												ControlId: "control-2",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{ControlID: "control-1", IncludeRules: []string{"*"}, ExcludeRules: []string{"rule-1"}},
					{ControlID: "control-2", IncludeRules: []string{"*"}},
				},
			},
			wantActivities: &[]oscalTypes.Activity{
				{
					Title: "rule-1",
					Props: &[]oscalTypes.Property{
						{
							Name:  "skipped",
							Value: "true",
							Ns:    extensions.TrestleNameSpace,
						},
					},
					RelatedControls: nil,
				},
				{
					Title: "rule-2",
					RelatedControls: &oscalTypes.ReviewedControls{
						ControlSelections: []oscalTypes.AssessedControls{
							{
								IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
									{
										ControlId: "control-2",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Success/MissingIncludeRules",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "rule-1",
							RelatedControls: &oscalTypes.ReviewedControls{
								ControlSelections: []oscalTypes.AssessedControls{
									{
										IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
											{
												ControlId: "control-1",
											},
										},
									},
								},
							},
						},
						{
							Title: "rule-2",
							RelatedControls: &oscalTypes.ReviewedControls{
								ControlSelections: []oscalTypes.AssessedControls{
									{
										IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
											{
												ControlId: "control-1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{ControlID: "control-1", ExcludeRules: []string{"rule-1"}}, // Missing includeRules should default to "*"
				},
			},
			wantActivities: &[]oscalTypes.Activity{
				{
					Title: "rule-1",
					Props: &[]oscalTypes.Property{
						{
							Name:  "skipped",
							Value: "true",
							Ns:    extensions.TrestleNameSpace,
						},
					},
					RelatedControls: nil,
				},
				{
					Title: "rule-2",
					RelatedControls: &oscalTypes.ReviewedControls{
						ControlSelections: []oscalTypes.AssessedControls{
							{
								IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
									{
										ControlId: "control-1",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Success/ExcludeAllRules",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "rule-1",
							RelatedControls: &oscalTypes.ReviewedControls{
								ControlSelections: []oscalTypes.AssessedControls{
									{
										IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
											{
												ControlId: "control-1",
											},
										},
									},
								},
							},
						},
						{
							Title: "rule-2",
							RelatedControls: &oscalTypes.ReviewedControls{
								ControlSelections: []oscalTypes.AssessedControls{
									{
										IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
											{
												ControlId: "control-1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID: "test",
				IncludeControls: []ControlEntry{
					{ControlID: "control-1", IncludeRules: []string{"rule-1", "rule-2"}, ExcludeRules: []string{"*"}}, // ExcludeRules="*" should override includeRules
				},
			},
			wantActivities: &[]oscalTypes.Activity{
				{
					Title: "rule-1",
					Props: &[]oscalTypes.Property{
						{
							Name:  "skipped",
							Value: "true",
							Ns:    extensions.TrestleNameSpace,
						},
					},
					RelatedControls: nil,
				},
				{
					Title: "rule-2",
					Props: &[]oscalTypes.Property{
						{
							Name:  "skipped",
							Value: "true",
							Ns:    extensions.TrestleNameSpace,
						},
					},
					RelatedControls: nil,
				},
			},
		},
		{
			name: "Success/GlobalExcludeOverridesInclude",
			basePlan: &oscalTypes.AssessmentPlan{
				LocalDefinitions: &oscalTypes.LocalDefinitions{
					Activities: &[]oscalTypes.Activity{
						{
							Title: "rule-1",
							RelatedControls: &oscalTypes.ReviewedControls{
								ControlSelections: []oscalTypes.AssessedControls{
									{
										IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
											{
												ControlId: "control-1",
											},
										},
									},
								},
							},
						},
						{
							Title: "rule-2",
							RelatedControls: &oscalTypes.ReviewedControls{
								ControlSelections: []oscalTypes.AssessedControls{
									{
										IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
											{
												ControlId: "control-1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			scope: AssessmentScope{
				FrameworkID:        "test",
				GlobalExcludeRules: []string{"rule-1"}, // Global exclude should override control-specific include
				IncludeControls: []ControlEntry{
					{ControlID: "control-1", IncludeRules: []string{"rule-1", "rule-2"}}, // Explicitly includes rule-1, but global exclude wins
				},
			},
			wantActivities: &[]oscalTypes.Activity{
				{
					Title: "rule-1",
					Props: &[]oscalTypes.Property{
						{
							Name:  "skipped",
							Value: "true",
							Ns:    extensions.TrestleNameSpace,
						},
					},
					RelatedControls: nil,
				},
				{
					Title: "rule-2",
					RelatedControls: &oscalTypes.ReviewedControls{
						ControlSelections: []oscalTypes.AssessedControls{
							{
								IncludeControls: &[]oscalTypes.AssessedControlsSelectControlById{
									{
										ControlId: "control-1",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := tt.scope
			err := scope.ApplyScope(tt.basePlan, testLogger)
			require.NoError(t, err)
			require.Equal(t, tt.wantActivities, tt.basePlan.LocalDefinitions.Activities)
		})
	}
}
