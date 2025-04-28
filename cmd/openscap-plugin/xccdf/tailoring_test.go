// SPDX-License-Identifier: Apache-2.0

package xccdf

import (
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/xccdf"
	"github.com/complytime/complytime/cmd/openscap-plugin/config"
	"github.com/oscal-compass/compliance-to-policy-go/v2/policy"
	"github.com/oscal-compass/oscal-sdk-go/extensions"
)

// This is a supporting function to get the profile element from the testing Datastream.
// It is used by TestGetTailoringSelections and TestGetTailoringValues.
func getProfileElementTest(t *testing.T, profileID string) (*xccdf.ProfileElement, error) {
	doc, _ := LoadDsTest(t, "ssg-rhel-ds.xml")
	dsProfile, err := getDsProfile(doc, profileID)
	if err != nil {
		t.Fatalf("failed to get profile: %v", err)
	}
	parsedProfile, err := initProfile(dsProfile, profileID)
	if err != nil {
		t.Fatalf("failed to init parsed profile: %v", err)
	}
	return parsedProfile, nil
}

// TestGetTailoringID tests the getTailoringID function.
func TestGetTailoringID(t *testing.T) {
	expected := "xccdf_complytime.openscapplugin_tailoring_complytime"
	result := getTailoringID()

	if result != expected {
		t.Errorf("getTailoringID() = %v; want %v", result, expected)
	}
}

// TestGetTailoringProfileID tests the getTailoringProfileID function.
func TestGetTailoringProfileID(t *testing.T) {
	profileId := "test_profile"
	expected := "xccdf_complytime.openscapplugin_profile_test_profile_complytime"
	result := getTailoringProfileID(profileId)

	if result != expected {
		t.Errorf("getTailoringProfileID(%v) = %v; want %v", profileId, result, expected)
	}
}

// TestGetTailoringProfileTitle tests the getTailoringProfileTitle function.
func TestGetTailoringProfileTitle(t *testing.T) {
	tests := []struct {
		profileTitle string
		expected     string
	}{
		{"Test Profile", "ComplyTime Tailoring Profile - Test Profile"},
		{"test_profile_id", "ComplyTime Tailoring Profile - test_profile_id"},
	}

	for _, tt := range tests {
		result := getTailoringProfileTitle(tt.profileTitle)
		if result != tt.expected {
			t.Errorf("getTailoringProfileTitle(%v) = %v; want %v", tt.profileTitle, result, tt.expected)
		}
	}
}

// TestGetTailoringVersion tests the getTailoringVersion function.
func TestGetTailoringVersion(t *testing.T) {
	result := getTailoringVersion()

	if result.Value != "1" {
		t.Errorf("getTailoringVersion().Value = %v; want %v", result.Value, "1")
	}

	_, err := time.Parse(time.RFC3339, result.Time)
	if err != nil {
		t.Errorf("getTailoringVersion().Time = %v; not in RFC3339 format", result.Time)
	}
}

// TestGetTailoringBenchmarkHref tests the getTailoringBenchmarkHref function.
func TestGetTailoringBenchmarkHref(t *testing.T) {
	dsPath := filepath.Join(testDataDir, "ssg-rhel-ds.xml")
	expected := xccdf.BenchmarkElement{
		Href: dsPath,
	}
	result := getTailoringBenchmarkHref(dsPath)

	if result != expected {
		t.Errorf("getTailoringBenchmarkHref(%v) = %v; want %v", dsPath, result, expected)
	}
}

// TestValidateRuleExistence tests the validateRuleExistence function.
func TestValidateRuleExistence(t *testing.T) {
	tests := []struct {
		name          string
		policyRuleID  string
		dsRules       []DsRules
		expectedExist bool
	}{
		{
			name:         "Rule exists",
			policyRuleID: "rule1",
			dsRules: []DsRules{
				{ID: "xccdf_org.ssgproject.content_rule_rule1"},
				{ID: "xccdf_org.ssgproject.content_rule_rule2"},
			},
			expectedExist: true,
		},
		{
			name:         "Rule does not exist",
			policyRuleID: "rule3",
			dsRules: []DsRules{
				{ID: "xccdf_org.ssgproject.content_rule_rule1"},
				{ID: "xccdf_org.ssgproject.content_rule_rule2"},
			},
			expectedExist: false,
		},
		{
			name:          "Empty dsRules",
			policyRuleID:  "rule1",
			dsRules:       []DsRules{},
			expectedExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateRuleExistence(tt.policyRuleID, tt.dsRules)
			if result != tt.expectedExist {
				t.Errorf("validateRuleExistence(%v, %v) = %v; want %v", tt.policyRuleID, tt.dsRules, result, tt.expectedExist)
			}
		})
	}
}

// TestValidateVariableExistence tests the validateVariableExistence function.
func TestValidateVariableExistence(t *testing.T) {
	tests := []struct {
		name              string
		policyVariableID  string
		dsVariables       []DsVariables
		expectedExistence bool
	}{
		{
			name:             "Variable exists",
			policyVariableID: "var1",
			dsVariables: []DsVariables{
				{ID: "xccdf_org.ssgproject.content_value_var1"},
				{ID: "xccdf_org.ssgproject.content_value_var2"},
			},
			expectedExistence: true,
		},
		{
			name:             "Variable does not exist",
			policyVariableID: "var3",
			dsVariables: []DsVariables{
				{ID: "xccdf_org.ssgproject.content_value_var1"},
				{ID: "xccdf_org.ssgproject.content_value_var2"},
			},
			expectedExistence: false,
		},
		{
			name:              "Empty dsVariables",
			policyVariableID:  "var1",
			dsVariables:       []DsVariables{},
			expectedExistence: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateVariableExistence(tt.policyVariableID, tt.dsVariables)
			if result != tt.expectedExistence {
				t.Errorf("validateVariableExistence(%v, %v) = %v; want %v", tt.policyVariableID, tt.dsVariables, result, tt.expectedExistence)
			}
		})
	}
}

// TestUnselectAbsentRules tests the unselectAbsentRules function.
func TestUnselectAbsentRules(t *testing.T) {
	tests := []struct {
		name                string
		tailoringSelections []xccdf.SelectElement
		dsProfileSelections []xccdf.SelectElement
		oscalPolicy         policy.Policy
		expectedSelections  []xccdf.SelectElement
	}{
		{
			name:                "No absent rules",
			tailoringSelections: []xccdf.SelectElement{},
			dsProfileSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule1", Selected: true},
				{IDRef: "xccdf_org.ssgproject.content_rule_rule2", Selected: true},
			},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{ID: "rule1"}},
				{Rule: extensions.Rule{ID: "rule2"}},
			},
			expectedSelections: []xccdf.SelectElement{},
		},
		{
			name:                "One absent rule",
			tailoringSelections: []xccdf.SelectElement{},
			dsProfileSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule1", Selected: true},
				{IDRef: "xccdf_org.ssgproject.content_rule_rule2", Selected: true},
			},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{ID: "rule1"}},
			},
			expectedSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule2", Selected: false},
			},
		},
		{
			name:                "All absent rules",
			tailoringSelections: []xccdf.SelectElement{},
			dsProfileSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule1", Selected: true},
				{IDRef: "xccdf_org.ssgproject.content_rule_rule2", Selected: true},
			},
			oscalPolicy: policy.Policy{},
			expectedSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule1", Selected: false},
				{IDRef: "xccdf_org.ssgproject.content_rule_rule2", Selected: false},
			},
		},
		{
			name:                "No dsProfileSelections",
			tailoringSelections: []xccdf.SelectElement{},
			dsProfileSelections: []xccdf.SelectElement{},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{ID: "rule1"}},
			},
			expectedSelections: []xccdf.SelectElement{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unselectAbsentRules(tt.tailoringSelections, tt.dsProfileSelections, tt.oscalPolicy)
			if len(result) != len(tt.expectedSelections) {
				t.Errorf("unselectAbsentRules() length = %v; want %v", len(result), len(tt.expectedSelections))
			}
			for i, selection := range result {
				if selection.IDRef != tt.expectedSelections[i].IDRef || selection.Selected != tt.expectedSelections[i].Selected {
					t.Errorf("unselectAbsentRules()[%d] = %v; want %v", i, selection, tt.expectedSelections[i])
				}
			}
		})
	}
}

// TestSelectAdditionalRules tests the selectAdditionalRules function.
func TestSelectAdditionalRules(t *testing.T) {
	tests := []struct {
		name                string
		tailoringSelections []xccdf.SelectElement
		dsProfileSelections []xccdf.SelectElement
		oscalPolicy         policy.Policy
		expectedSelections  []xccdf.SelectElement
	}{
		{
			name:                "No additional rules",
			tailoringSelections: []xccdf.SelectElement{},
			dsProfileSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule1", Selected: true},
				{IDRef: "xccdf_org.ssgproject.content_rule_rule2", Selected: true},
			},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{ID: "rule1"}},
				{Rule: extensions.Rule{ID: "rule2"}},
			},
			expectedSelections: []xccdf.SelectElement{},
		},
		{
			name:                "One additional rule",
			tailoringSelections: []xccdf.SelectElement{},
			dsProfileSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule1", Selected: true},
			},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{ID: "rule1"}},
				{Rule: extensions.Rule{ID: "rule2"}},
			},
			expectedSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule2", Selected: true},
			},
		},
		{
			name:                "All additional rules",
			tailoringSelections: []xccdf.SelectElement{},
			dsProfileSelections: []xccdf.SelectElement{},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{ID: "rule1"}},
				{Rule: extensions.Rule{ID: "rule2"}},
			},
			expectedSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule1", Selected: true},
				{IDRef: "xccdf_org.ssgproject.content_rule_rule2", Selected: true},
			},
		},
		{
			name:                "Rule already in dsProfile but unselected",
			tailoringSelections: []xccdf.SelectElement{},
			dsProfileSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule1", Selected: false},
			},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{ID: "rule1"}},
			},
			expectedSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule1", Selected: true},
			},
		},
		{
			name:                "One additional rule informed twice",
			tailoringSelections: []xccdf.SelectElement{},
			dsProfileSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule1", Selected: true},
			},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{ID: "rule1"}},
				{Rule: extensions.Rule{ID: "rule2"}},
				{Rule: extensions.Rule{ID: "rule2"}},
			},
			expectedSelections: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_rule2", Selected: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectAdditionalRules(tt.tailoringSelections, tt.dsProfileSelections, tt.oscalPolicy)
			if len(result) != len(tt.expectedSelections) {
				t.Errorf("selectAdditionalRules() length = %v; want %v", len(result), len(tt.expectedSelections))
			}
			for i, selection := range result {
				if selection.IDRef != tt.expectedSelections[i].IDRef || selection.Selected != tt.expectedSelections[i].Selected {
					t.Errorf("selectAdditionalRules()[%d] = %v; want %v", i, selection, tt.expectedSelections[i])
				}
			}
		})
	}
}

// TestGetTailoringSelections tests the getTailoringSelections function.
func TestGetTailoringSelections(t *testing.T) {
	dsPath := filepath.Join(testDataDir, "ssg-rhel-ds.xml")
	parsedProfile, _ := getProfileElementTest(t, "xccdf_org.ssgproject.content_profile_test_profile")

	tests := []struct {
		name           string
		oscalPolicy    policy.Policy
		expectedError  bool
		expectedResult []xccdf.SelectElement
	}{
		{
			name: "All rules present",
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{ID: "package_telnet-server_removed"}},
				{Rule: extensions.Rule{ID: "package_telnet_removed"}},
				{Rule: extensions.Rule{ID: "set_password_hashing_algorithm_logindefs"}},
				{Rule: extensions.Rule{ID: "set_password_hashing_algorithm_systemauth"}},
			},
			expectedError:  false,
			expectedResult: []xccdf.SelectElement{},
		},
		{
			name: "One rule missing in datastream",
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{ID: "package_telnet-server_removed"}},
				{Rule: extensions.Rule{ID: "package_telnet_removed"}},
				{Rule: extensions.Rule{ID: "set_password_hashing_algorithm_logindefs"}},
				{Rule: extensions.Rule{ID: "set_password_hashing_algorithm_systemauth"}},
				{Rule: extensions.Rule{ID: "this_rule_is_not_in_datastream"}},
			},
			expectedError:  true,
			expectedResult: nil,
		},
		{
			name:        "No rules in OSCAL policy",
			oscalPolicy: policy.Policy{},
			expectedResult: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_package_telnet-server_removed", Selected: false},
				{IDRef: "xccdf_org.ssgproject.content_rule_package_telnet_removed", Selected: false},
				{IDRef: "xccdf_org.ssgproject.content_rule_set_password_hashing_algorithm_logindefs", Selected: false},
				{IDRef: "xccdf_org.ssgproject.content_rule_set_password_hashing_algorithm_systemauth", Selected: false},
			},
		},
		{
			name: "Additional rule in OSCAL policy",
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{ID: "package_telnet-server_removed"}},
				{Rule: extensions.Rule{ID: "package_telnet_removed"}},
				{Rule: extensions.Rule{ID: "set_password_hashing_algorithm_logindefs"}},
				{Rule: extensions.Rule{ID: "set_password_hashing_algorithm_systemauth"}},
				{Rule: extensions.Rule{ID: "account_unique_id"}},
			},
			expectedError: false,
			expectedResult: []xccdf.SelectElement{
				{IDRef: "xccdf_org.ssgproject.content_rule_account_unique_id", Selected: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getTailoringSelections(tt.oscalPolicy, parsedProfile, dsPath)
			if (err != nil) != tt.expectedError {
				t.Errorf("getTailoringSelections() error = %v; want %v", err, tt.expectedError)
			}
			if len(result) != len(tt.expectedResult) {
				t.Errorf("getTailoringSelections() length = %v; want %v", len(result), len(tt.expectedResult))
			}
			for i, selection := range result {
				if selection.IDRef != tt.expectedResult[i].IDRef || selection.Selected != tt.expectedResult[i].Selected {
					t.Errorf("getTailoringSelections()[%d] = %v; want %v", i, selection, tt.expectedResult[i])
				}
			}
		})
	}
}

// TestUpdateTailoringValues tests the updateTailoringValues function.
func TestUpdateTailoringValues(t *testing.T) {
	tests := []struct {
		name            string
		tailoringValues []xccdf.SetValueElement
		dsProfileValues []xccdf.SetValueElement
		oscalPolicy     policy.Policy
		expectedValues  []xccdf.SetValueElement
	}{
		{
			name:            "No additional values",
			tailoringValues: []xccdf.SetValueElement{},
			dsProfileValues: []xccdf.SetValueElement{
				{IDRef: "xccdf_org.ssgproject.content_value_var1", Value: "value1"},
				{IDRef: "xccdf_org.ssgproject.content_value_var2", Value: "value2"},
			},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var1", Value: "value1"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var2", Value: "value2"}}},
			},
			expectedValues: []xccdf.SetValueElement{},
		},
		{
			name:            "One additional value",
			tailoringValues: []xccdf.SetValueElement{},
			dsProfileValues: []xccdf.SetValueElement{
				{IDRef: "xccdf_org.ssgproject.content_value_var1", Value: "value1"},
			},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var1", Value: "value1"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var2", Value: "value2"}}},
			},
			expectedValues: []xccdf.SetValueElement{
				{IDRef: "xccdf_org.ssgproject.content_value_var2", Value: "value2"},
			},
		},
		{
			name:            "All additional values",
			tailoringValues: []xccdf.SetValueElement{},
			dsProfileValues: []xccdf.SetValueElement{},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var1", Value: "value1"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var2", Value: "value2"}}},
			},
			expectedValues: []xccdf.SetValueElement{
				{IDRef: "xccdf_org.ssgproject.content_value_var1", Value: "value1"},
				{IDRef: "xccdf_org.ssgproject.content_value_var2", Value: "value2"},
			},
		},
		{
			name:            "Variable already in dsProfile but different value",
			tailoringValues: []xccdf.SetValueElement{},
			dsProfileValues: []xccdf.SetValueElement{
				{IDRef: "xccdf_org.ssgproject.content_value_var1", Value: "old_value"},
			},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var1", Value: "new_value"}}},
			},
			expectedValues: []xccdf.SetValueElement{
				{IDRef: "xccdf_org.ssgproject.content_value_var1", Value: "new_value"},
			},
		},
		{
			name:            "Rule without parameter",
			tailoringValues: []xccdf.SetValueElement{},
			dsProfileValues: []xccdf.SetValueElement{
				{IDRef: "xccdf_org.ssgproject.content_value_var1", Value: "value1"},
			},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{Parameter: nil}},
			},
			expectedValues: []xccdf.SetValueElement{},
		},
		{
			name:            "One additional value informed twice",
			tailoringValues: []xccdf.SetValueElement{},
			dsProfileValues: []xccdf.SetValueElement{
				{IDRef: "xccdf_org.ssgproject.content_value_var1", Value: "value1"},
			},
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var1", Value: "value1"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var2", Value: "value2"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var2", Value: "value2"}}},
			},
			expectedValues: []xccdf.SetValueElement{
				{IDRef: "xccdf_org.ssgproject.content_value_var2", Value: "value2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updateTailoringValues(tt.tailoringValues, tt.dsProfileValues, tt.oscalPolicy)
			if len(result) != len(tt.expectedValues) {
				t.Errorf("updateTailoringValues() length = %v; want %v", len(result), len(tt.expectedValues))
			}
			for i, value := range result {
				if value.IDRef != tt.expectedValues[i].IDRef || value.Value != tt.expectedValues[i].Value {
					t.Errorf("updateTailoringValues()[%d] = %v; want %v", i, value, tt.expectedValues[i])
				}
			}
		})
	}
}

// TestGetTailoringValues tests the getTailoringValues function.
func TestGetTailoringValues(t *testing.T) {
	dsPath := filepath.Join(testDataDir, "ssg-rhel-ds.xml")

	tests := []struct {
		name           string
		oscalPolicy    policy.Policy
		expectedError  bool
		expectedResult []xccdf.SetValueElement
	}{
		{
			name: "All variables present",
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_hashing_algorithm", Value: "SHA512"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_hashing_algorithm_pam", Value: "sha512"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_accounts_tmout", Value: "900"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_pam_remember_control_flag", Value: "requisite,required"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_pam_remember", Value: "5"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_system_crypto_policy", Value: "DEFAULT"}}},
			},
			expectedError:  false,
			expectedResult: []xccdf.SetValueElement{},
		},
		{
			name: "One variable missing in datastream",
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_hashing_algorithm", Value: "SHA512"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_hashing_algorithm_pam", Value: "sha512"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_accounts_tmout", Value: "900"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_pam_remember_control_flag", Value: "requisite,required"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_pam_remember", Value: "5"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "this_variable_is_not_in_datastream", Value: "value"}}},
			},
			expectedError:  true,
			expectedResult: nil,
		},
		{
			name: "Additional variable in OSCAL policy",
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_hashing_algorithm", Value: "SHA512"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_hashing_algorithm_pam", Value: "sha512"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_accounts_tmout", Value: "900"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_pam_remember_control_flag", Value: "requisite,required"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_password_pam_remember", Value: "5"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_system_crypto_policy", Value: "DEFAULT"}}},
				{Rule: extensions.Rule{Parameter: &extensions.Parameter{ID: "var_selinux_policy_name", Value: "mls"}}},
			},
			expectedError: false,
			expectedResult: []xccdf.SetValueElement{
				{IDRef: "xccdf_org.ssgproject.content_value_var_selinux_policy_name", Value: "mls"},
			},
		},
		{
			name: "OSCAL policy without variables",
			oscalPolicy: policy.Policy{
				{Rule: extensions.Rule{Parameter: nil}},
			},
			expectedError:  false,
			expectedResult: []xccdf.SetValueElement{},
		},
	}

	for _, tt := range tests {
		// Variables options are resolved during the process, so we need to get the profile element again.
		parsedProfile, _ := getProfileElementTest(t, "xccdf_org.ssgproject.content_profile_test_profile")

		t.Run(tt.name, func(t *testing.T) {
			result, err := getTailoringValues(tt.oscalPolicy, parsedProfile, dsPath)
			if (err != nil) != tt.expectedError {
				t.Errorf("getTailoringValues() error = %v; want %v", err, tt.expectedError)
			}
			if len(result) != len(tt.expectedResult) {
				t.Errorf("getTailoringValues() length = %v; want %v", len(result), len(tt.expectedResult))
			}
			for i, value := range result {
				if value.IDRef != tt.expectedResult[i].IDRef || value.Value != tt.expectedResult[i].Value {
					t.Errorf("getTailoringValues()[%d] = %v; want %v", i, value, tt.expectedResult[i])
				}
			}
		})
	}
}

// TestGetTailoringProfile tests the getTailoringProfile function.
func TestGetTailoringProfile(t *testing.T) {
	dsPath := filepath.Join(testDataDir, "ssg-rhel-ds.xml")
	profileId := "test_profile"

	tailoringPolicy := policy.Policy{
		{
			Rule: extensions.Rule{
				ID:          "set_password_hashing_algorithm_logindefs",
				Description: "Set Password Hashing Algorithm in /etc/login.defs",
				Parameter: &extensions.Parameter{
					ID:          "var_password_hashing_algorithm",
					Description: "Password Hashing algorithm",
					Value:       "YESCRYPT",
				},
			},
		},
	}

	expected := xccdf.ProfileElement{
		ID: getTailoringProfileID(profileId),
		Title: &xccdf.TitleOrDescriptionElement{
			Value: "ComplyTime Tailoring Profile - Test Profile",
		},
		Selections: []xccdf.SelectElement{
			{IDRef: "xccdf_org.ssgproject.content_rule_package_telnet-server_removed", Selected: false},
			{IDRef: "xccdf_org.ssgproject.content_rule_package_telnet_removed", Selected: false},
			{IDRef: "xccdf_org.ssgproject.content_rule_set_password_hashing_algorithm_systemauth", Selected: false},
		},
		Values: []xccdf.SetValueElement{
			{IDRef: "xccdf_org.ssgproject.content_value_var_password_hashing_algorithm", Value: "YESCRYPT"},
		},
	}

	result, err := getTailoringProfile(profileId, dsPath, tailoringPolicy)
	if err != nil {
		t.Fatalf("getTailoringProfile() error = %v", err)
	}

	if result.ID != expected.ID {
		t.Errorf("getTailoringProfile().ID = %v; want %v", result.ID, expected.ID)
	}
	if result.Title.Value != expected.Title.Value {
		t.Errorf("getTailoringProfile().Title.Value = %v; want %v", result.Title.Value, expected.Title.Value)
	}
	if len(result.Selections) != len(expected.Selections) {
		t.Errorf("getTailoringProfile().Selections = %v; want %v", result.Selections, expected.Selections)
	}
	if len(result.Values) != len(expected.Values) {
		t.Errorf("getTailoringProfile().Values = %v; want %v", result.Values, expected.Values)
	}
}

// This is a supporting function used by TestPolicyToXML.
// It removes the time attribute from the XML because it is generated
// dynamically and may differ in some seconds during the tests.
func removeVersionTimeTest(xml string) string {
	re := regexp.MustCompile(`time="[^"]*"`)
	return re.ReplaceAllString(xml, `time=""`)
}

// TestPolicyToXML tests the PolicyToXML function.
func TestPolicyToXML(t *testing.T) {
	dsPath := filepath.Join(testDataDir, "ssg-rhel-ds.xml")
	profileId := "test_profile"

	tailoringPolicy := policy.Policy{
		{
			Rule: extensions.Rule{
				ID:          "account_unique_id",
				Description: "Ensure All Accounts on the System Have Unique User IDs",
				Parameter: &extensions.Parameter{
					ID:          "var_password_hashing_algorithm",
					Description: "Password Hashing algorithm",
					Value:       "YESCRYPT",
				},
			},
		},
	}

	cfg := new(config.Config)
	cfg.Files.Datastream = dsPath
	cfg.Parameters.Profile = profileId

	expectedXML := `<?xml version="1.0" encoding="UTF-8"?>
<xccdf-1.2:Tailoring xmlns:xccdf-1.2="http://checklists.nist.gov/xccdf/1.2" id="xccdf_complytime.openscapplugin_tailoring_complytime">
  <xccdf-1.2:benchmark href="` + dsPath + `"></xccdf-1.2:benchmark>
  <xccdf-1.2:version time="` + getTailoringVersion().Time + `">1</xccdf-1.2:version>
  <xccdf-1.2:Profile id="xccdf_complytime.openscapplugin_profile_test_profile_complytime" extends="xccdf_org.ssgproject.content_profile_test_profile">
    <xccdf-1.2:title override="true">ComplyTime Tailoring Profile - Test Profile</xccdf-1.2:title>
    <xccdf-1.2:select idref="xccdf_org.ssgproject.content_rule_package_telnet-server_removed" selected="false"></xccdf-1.2:select>
    <xccdf-1.2:select idref="xccdf_org.ssgproject.content_rule_package_telnet_removed" selected="false"></xccdf-1.2:select>
    <xccdf-1.2:select idref="xccdf_org.ssgproject.content_rule_set_password_hashing_algorithm_logindefs" selected="false"></xccdf-1.2:select>
    <xccdf-1.2:select idref="xccdf_org.ssgproject.content_rule_set_password_hashing_algorithm_systemauth" selected="false"></xccdf-1.2:select>
    <xccdf-1.2:select idref="xccdf_org.ssgproject.content_rule_account_unique_id" selected="true"></xccdf-1.2:select>
    <xccdf-1.2:set-value idref="xccdf_org.ssgproject.content_value_var_password_hashing_algorithm">YESCRYPT</xccdf-1.2:set-value>
  </xccdf-1.2:Profile>
</xccdf-1.2:Tailoring>`

	result, err := PolicyToXML(tailoringPolicy, cfg)
	if err != nil {
		t.Fatalf("PolicyToXML() error = %v", err)
	}

	// It takes some seconds to generate the tailoring file and the time differs.
	// So we remove the time attribute to compare the XMLs.
	expected := removeVersionTimeTest(expectedXML)
	actual := removeVersionTimeTest(result)

	if actual != expected {
		t.Errorf("PolicyToXML() = %v; want %v", actual, expected)
	}
}
