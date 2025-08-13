// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"sort"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	"github.com/stretchr/testify/require"
)

func TestRenderKeyValuePair(t *testing.T) {

	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "Valid/RenderKeyValuePair",
			key:      "Control ID",
			value:    "r31",
			expected: "Control ID : r31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderKeyValuePair(tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetControlRulesColumnsAndRows(t *testing.T) {

	tests := []struct {
		name                 string
		control              control
		expectedColumnTitles []string
		expectedRows         []table.Row
	}{
		{
			name: "Valid/ControlWithRulesList",
			control: control{
				ID: "test-control-id",
				Rules: []rule{
					{ID: "rule-1", Plugin: "Plugin 1"},
					{ID: "rule-2", Plugin: "Plugin 2"},
				},
			},
			expectedColumnTitles: []string{"Rules In Control", "Plugin Used"},
			expectedRows: []table.Row{
				{"rule-1", "Plugin 1"},
				{"rule-2", "Plugin 2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columns, rows := getControlRulesColumnsAndRows(tt.control)
			require.Equal(t, len(columns), len(tt.expectedColumnTitles))
			require.Equal(t, len(rows), len(tt.expectedRows))

			sort.Slice(rows, func(i, j int) bool {
				return rows[i][0] < rows[j][0]
			})
			require.Equal(t, rows, tt.expectedRows)
		})
	}
}

func TestGetControlListColumnsAndRows(t *testing.T) {

	tests := []struct {
		name                 string
		controls             []control
		expectedColumnTitles []string
		expectedRows         []table.Row
	}{
		{
			name: "Valid/ControlsList",
			controls: []control{
				{
					ID:                   "test_control_id",
					Title:                "Test Control Title",
					ImplementationStatus: "implemented",
					Rules: []rule{
						{ID: "rule-1", Plugin: "plugin-1"},
						{ID: "rule-2", Plugin: "plugin-1"},
						{ID: "rule-3", Plugin: "plugin-2"},
					},
				},
			},
			expectedColumnTitles: []string{"Control ID", "Control Title", "Status", "Plugins Used"},
			expectedRows: []table.Row{
				{"test_control_id", "Test Control Title", "implemented", "plugin-1, plugin-2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columns, rows := getControlListColumnsAndRows(tt.controls)
			require.Equal(t, len(columns), len(tt.expectedColumnTitles))
			require.Equal(t, len(rows), len(tt.expectedRows))

			sort.Slice(rows, func(i, j int) bool {
				return rows[i][0] < rows[j][0]
			})
			require.Equal(t, rows, tt.expectedRows)
		})
	}
}

func TestGetRuleParametersColumnsAndRows(t *testing.T) {

	tests := []struct {
		name                 string
		ruleInfo             rule
		setParameters        map[string][]string
		expectedColumnTitles []string
		expectedRows         []table.Row
	}{
		{
			name: "Valid/RuleParametersList",
			ruleInfo: rule{
				ID:          "rule-1",
				Description: "Rule 1",
				Parameters: []parameter{
					{ID: "param-1", Description: "Parameter 1 description"},
					{ID: "param-2", Description: "Parameter 2 description"},
				},
			},
			setParameters: map[string][]string{
				"param-1": []string{"param-1-value"},
				"param-2": []string{"param-2-value-1", "param-2-value-2"},
			},
			expectedColumnTitles: []string{"Parameter ID", "Parameter Description", "Set  Value(s)"},
			expectedRows: []table.Row{
				{"param-1", "Parameter 1 description", "param-1-value"},
				{"param-2", "Parameter 2 description", "param-2-value-1, param-2-value-2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columns, rows := getRuleParametersColumnsAndRows(tt.ruleInfo, tt.setParameters)
			require.Equal(t, len(columns), len(tt.expectedColumnTitles))
			require.Equal(t, len(rows), len(tt.expectedRows))

			sort.Slice(rows, func(i, j int) bool {
				return rows[i][0] < rows[j][0]
			})
			require.Equal(t, rows, tt.expectedRows)
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name          string
		inputSlice    []string
		expectedSlice []string
	}{
		{
			name:          "Valid/SliceWithDuplicates",
			inputSlice:    []string{"one", "one", "two", "two"},
			expectedSlice: []string{"one", "two"},
		},
		{
			name:          "Valid/SliceWithDuplicates",
			inputSlice:    []string{},
			expectedSlice: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputSlice := removeDuplicates(tt.inputSlice)
			require.Equal(t, outputSlice, tt.expectedSlice)
		})
	}
}

func TestCalculateRowLimit(t *testing.T) {

	tests := []struct {
		name             string
		rowLimit         int
		availableRows    int
		expectedRowLimit int
	}{
		{
			name:             "Valid/DefaultRowLimit",
			rowLimit:         0,
			availableRows:    10,
			expectedRowLimit: 11,
		},
		{
			name:             "Valid/RowLimitLessThanAvailable",
			rowLimit:         5,
			availableRows:    10,
			expectedRowLimit: 6,
		},
		{
			name:             "Valid/RowLimitGreaterThanAvailable",
			rowLimit:         15,
			availableRows:    10,
			expectedRowLimit: 11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rowLimit := calculateRowLimit(tt.rowLimit, tt.availableRows)
			require.Equal(t, rowLimit, tt.expectedRowLimit)
		})
	}
}
