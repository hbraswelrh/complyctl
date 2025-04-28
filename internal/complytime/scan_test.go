// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"os"
	"path/filepath"
	"testing"

	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-2"
	"github.com/oscal-compass/oscal-sdk-go/models"
	"github.com/oscal-compass/oscal-sdk-go/validation"
	"github.com/stretchr/testify/require"
)

func TestWriteAssessmentResults(t *testing.T) {

	tmpDir := t.TempDir()
	testResultsPath := filepath.Join(tmpDir, "assessment-results.json")

	testAssessmentResults := oscalTypes.AssessmentResults{
		UUID: "228ff6d0-0d67-4c15-9c16-ece9a554c4de",
		ImportAp: oscalTypes.ImportAp{
			Href:    "https:/...",
			Remarks: "test",
		},
		Metadata: oscalTypes.Metadata{
			Title:        "example",
			OscalVersion: "1.1.2",
			Version:      "1.0.0",
		},
		Results: []oscalTypes.Result{
			{
				UUID:  "348fc6d0-706d-4c15-9c16-bce2a22ac3ee",
				Title: "test",
			},
		},
	}

	err := WriteAssessmentResults(&testAssessmentResults, testResultsPath)
	require.NoError(t, err)

	file, err := os.Open(testResultsPath)
	require.NoError(t, err)

	loadedAssessmentResults, err := models.NewAssessmentResults(file, validation.NoopValidator{})
	require.NoError(t, err)
	require.Equal(t, loadedAssessmentResults.Metadata.Title, testAssessmentResults.Metadata.Title)
}
