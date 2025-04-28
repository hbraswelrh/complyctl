// SPDX-License-Identifier: Apache-2.0

package terminal

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/complytime/complytime/internal/complytime"
)

func TestShowDefinitionTable(t *testing.T) {
	tests := []struct {
		name       string
		pretty     bool
		frameworks []complytime.Framework
		wantView   string
	}{
		{
			name: "Valid/PlainMode",
			frameworks: []complytime.Framework{
				{
					ID:                  "anotherexample",
					Title:               "Example Profile (moderate)",
					SupportedComponents: []string{"My Software"},
				},
				{
					ID:                  "example",
					Title:               "Example Profile (low)",
					SupportedComponents: []string{"My Software"},
				},
			},
			wantView: plainTable,
		},
		{
			name: "Valid/PrettyMode",
			frameworks: []complytime.Framework{
				{
					ID:                  "anotherexample",
					Title:               "Example Profile (moderate)",
					SupportedComponents: []string{"My Software"},
				},
				{
					ID:                  "example",
					Title:               "Example Profile (low)",
					SupportedComponents: []string{"My Software"},
				},
			},
			pretty:   true,
			wantView: populatedTable,
		},
		{
			name:       "Valid/EmptyPrettyMode",
			frameworks: []complytime.Framework{},
			pretty:     true,
			wantView:   emptyTable,
		},
		{
			name: "Valid/LongTitlePrettyMode",
			frameworks: []complytime.Framework{
				{
					ID:                  "anotherexample",
					Title:               "This is a very very very long title (moderate)",
					SupportedComponents: []string{"My Software"},
				},
			},
			pretty:   true,
			wantView: longTitle,
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			var gotView string
			if !c.pretty {
				out := bytes.NewBuffer(nil)
				ShowDefinitionTable(out, c.frameworks)
				gotView = out.String()
			} else {
				initialModel := ShowPrettyDefinitionTable(c.frameworks)
				gotView = initialModel.View()
			}
			require.Equal(t, c.wantView, gotView)
		})
	}
}

var (
	emptyTable = `┌──────────────────────────────────────────────────────────────────────────────────────┐
│ Title                           Framework ID          Supported Components           │
│──────────────────────────────────────────────────────────────────────────────────────│
│                                                                                      │
│                                                                                      │
│                                                                                      │
│                                                                                      │
│                                                                                      │
│                                                                                      │
└──────────────────────────────────────────────────────────────────────────────────────┘

Choose an option from the Framework ID column to use with complytime plan.
`
	populatedTable = `┌──────────────────────────────────────────────────────────────────────────────────────┐
│ Title                           Framework ID          Supported Components           │
│──────────────────────────────────────────────────────────────────────────────────────│
│ Example Profile (moderate)      anotherexample        My Software                    │
│ Example Profile (low)           example               My Software                    │
│                                                                                      │
│                                                                                      │
│                                                                                      │
│                                                                                      │
└──────────────────────────────────────────────────────────────────────────────────────┘

Choose an option from the Framework ID column to use with complytime plan.
`
	longTitle = `┌──────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Title                                           Framework ID          Supported Components           │
│──────────────────────────────────────────────────────────────────────────────────────────────────────│
│ This is a very very very long title (moderate)  anotherexample        My Software                    │
│                                                                                                      │
│                                                                                                      │
│                                                                                                      │
│                                                                                                      │
│                                                                                                      │
└──────────────────────────────────────────────────────────────────────────────────────────────────────┘

Choose an option from the Framework ID column to use with complytime plan.
`
	plainTable = "Title                         Framework ID        Supported Components          \n" +
		"Example Profile (moderate)    anotherexample      My Software                   \n" +
		"Example Profile (low)         example             My Software                   \n"
)
