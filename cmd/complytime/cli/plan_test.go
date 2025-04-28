// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"testing"

	"github.com/oscal-compass/oscal-sdk-go/validation"
	"github.com/stretchr/testify/require"

	"github.com/complytime/complytime/cmd/complytime/option"
)

func TestPlansInWorkspace(t *testing.T) {
	// Test that the proper error messages are thrown when
	// working with assessment plans in the user workspace.

	testOpts := &option.ComplyTime{
		UserWorkspace: "doesnotexist",
	}
	wantErr := "error: assessment plan does not exist in workspace doesnotexist: o" +
		"pen doesnotexist/assessment-plan.json: no such file or directory\n\nDid you run the plan command?"
	_, _, gotErr := loadPlan(testOpts, validation.NoopValidator{})
	require.EqualError(t, gotErr, wantErr)

	testOpts.UserWorkspace = "testdata"
	plan, gotPath, err := loadPlan(testOpts, validation.NoopValidator{})
	require.NoError(t, err)
	require.Equal(t, "testdata/assessment-plan.json", gotPath)
	wantErr = "assessment plan in \"testdata\" workspace does not have associated activities: no local activities detected"
	_, gotErr = getPlanSettings(testOpts, plan)
	require.EqualError(t, gotErr, wantErr)
}
