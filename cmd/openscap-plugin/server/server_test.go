package server

import (
	"errors"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/oscal-compass/compliance-to-policy-go/v2/policy"
	"github.com/stretchr/testify/assert"
)

func TestMapResultStatus(t *testing.T) {
	tests := []struct {
		name           string
		xmlContent     string
		expectedResult policy.Result
		expectedError  error
	}{
		{
			name:           "Pass result",
			xmlContent:     `<rule-result><result>pass</result></rule-result>`,
			expectedResult: policy.ResultPass,
			expectedError:  nil,
		},
		{
			name:           "Fail result",
			xmlContent:     `<rule-result><result>fail</result></rule-result>`,
			expectedResult: policy.ResultFail,
			expectedError:  nil,
		},
		{
			name:           "Not selected result",
			xmlContent:     `<rule-result><result>notselected</result></rule-result>`,
			expectedResult: policy.ResultError,
			expectedError:  nil,
		},
		{
			name:           "Not selected result",
			xmlContent:     `<rule-result><result>notapplicable</result></rule-result>`,
			expectedResult: policy.ResultError,
			expectedError:  nil,
		},
		{
			name:           "Error result",
			xmlContent:     `<rule-result><result>error</result></rule-result>`,
			expectedResult: policy.ResultError,
			expectedError:  nil,
		},
		{
			name:           "Unknown result",
			xmlContent:     `<rule-result><result>unknown</result></rule-result>`,
			expectedResult: policy.ResultError,
			expectedError:  nil,
		},
		{
			name:           "Invalid result",
			xmlContent:     `<rule-result><result>invalid</result></rule-result>`,
			expectedResult: policy.ResultInvalid,
			expectedError:  errors.New("couldn't match invalid"),
		},
		{
			name:           "No result element",
			xmlContent:     `<rule-result></rule-result>`,
			expectedResult: policy.ResultInvalid,
			expectedError:  errors.New("result node has no 'result' attribute"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := xmlquery.Parse(strings.NewReader(tt.xmlContent))
			assert.NoError(t, err)

			result, err := mapResultStatus(node.SelectElement("rule-result"))
			assert.Equal(t, tt.expectedResult, result)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseCheck(t *testing.T) {
	tests := []struct {
		name           string
		xmlContent     string
		expectedResult string
		expectedError  error
	}{
		{
			name:           "Valid/ExpectedFormat",
			xmlContent:     `<check-content-ref name="oval:ssg-audit_perm_change_success:def:1"/>`,
			expectedResult: "audit_perm_change_success",
		},
		{
			name:           "Invalid/UnexpectedFormat",
			xmlContent:     `<check-content-ref name="ovalssg-audit_perm_change_success:def:1"/>`,
			expectedResult: "",
			expectedError:  errors.New("check id \"ovalssg-audit_perm_change_success:def:1\" is in unexpected format"),
		},
		{
			name:           "Invalid/NoNameAttribute",
			xmlContent:     `<check-content-ref/>`,
			expectedResult: "",
			expectedError:  errors.New("check-content-ref node has no 'name' attribute"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := xmlquery.Parse(strings.NewReader(tt.xmlContent))
			assert.NoError(t, err)
			check, err := parseCheck(node.SelectElement("check-content-ref"))
			assert.Equal(t, tt.expectedResult, check)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
