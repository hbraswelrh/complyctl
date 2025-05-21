// SPDX-License-Identifier: Apache-2.0

package plan

import (
	"path/filepath"
	"testing"

	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/oscal-compass/oscal-sdk-go/extensions"
	"github.com/oscal-compass/oscal-sdk-go/validation"
	"github.com/stretchr/testify/require"
)

func TestPlan(t *testing.T) {
	tmpDir := t.TempDir()
	testPlanPath := filepath.Join(tmpDir, "assessment-plan.json")

	// Testing reading and writing Assessment plan workflows

	// Test Write -> Read -> Settings with errors
	testPlan := oscalTypes.AssessmentPlan{
		UUID: "228ff6d0-0d67-4c15-9c16-ece9a554c4de",
		Metadata: oscalTypes.Metadata{
			Title:        "example",
			OscalVersion: "1.1.2",
			Version:      "1.0.0",
		},
	}

	err := WritePlan(&testPlan, "testid", testPlanPath)
	require.NoError(t, err)

	ap, err := ReadPlan(testPlanPath, validation.NoopValidator{})
	require.NoError(t, err)
	require.NotNil(t, ap)

	_, err = Settings(ap)
	require.ErrorIs(t, err, ErrNoActivities)

	// Test Write -> Read -> Settings on a happy path
	localDefs := oscalTypes.LocalDefinitions{
		Activities: &[]oscalTypes.Activity{
			{
				Description: "activity",
				Title:       "my-activity",
				UUID:        "228ff6d0-0d67-4c15-9c16-ece9a554c4df",
			},
		},
	}
	testPlan.LocalDefinitions = &localDefs

	err = WritePlan(&testPlan, "testid", testPlanPath)
	require.NoError(t, err)

	// read plan to ensure it has the expected props
	ap, err = ReadPlan(testPlanPath, validation.NoopValidator{})
	require.NoError(t, err)
	require.NotNil(t, ap)

	_, err = Settings(ap)
	require.NoError(t, err)

	wantProp := oscalTypes.Property{
		Name:  extensions.FrameworkProp,
		Value: "testid",
		Ns:    extensions.TrestleNameSpace,
	}
	require.NotNil(t, ap.Metadata.Props)
	require.Contains(t, *ap.Metadata.Props, wantProp)
}
