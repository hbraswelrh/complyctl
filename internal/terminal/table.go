package terminal

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/complytime/complytime/internal/complytime"
)

// ShowDefinitionTable prints a plain table with given framework data.
func ShowDefinitionTable(writer io.Writer, frameworks []complytime.Framework) {
	columns, rows := getColumnsAndRows(frameworks)
	for _, col := range columns {
		_, _ = fmt.Fprintf(writer, "%-*s", col.Width, col.Title)
	}
	_, _ = fmt.Fprintln(writer)
	for _, row := range rows {
		for i, cell := range row {
			_, _ = fmt.Fprintf(writer, "%-*s", columns[i].Width, cell)
		}
		_, _ = fmt.Fprintln(writer)
	}

}

// ShowPrettyDefinitionTable returns a Model to be used with a `bubbletea` Program that
// renders a table with given Framework data.
func ShowPrettyDefinitionTable(frameworks []complytime.Framework) tea.Model {
	columns, rows := getColumnsAndRows(frameworks)
	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	tableStyle := table.DefaultStyles()
	tableStyle.Header = tableStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	tableStyle.Cell = tableStyle.Cell.
		Foreground(lipgloss.Color("250")).
		Bold(true)
	tbl.SetStyles(tableStyle)

	return model{tbl}
}

// getColumnsAndRows returns populate columns and row for printing tables.
func getColumnsAndRows(frameworks []complytime.Framework) ([]table.Column, []table.Row) {
	var rows []table.Row
	for _, framework := range frameworks {
		row := table.Row{framework.Title, framework.ID, strings.Join(framework.SupportedComponents, ", ")}
		rows = append(rows, row)
	}
	// Sort the rows slice by the framework short name
	sort.SliceStable(rows, func(i, j int) bool { return rows[i][1] < rows[j][1] })

	// Set columns with default widths
	columns := []table.Column{
		{Title: "Title", Width: 30},
		{Title: "Framework ID", Width: 20},
		{Title: "Supported Components", Width: 30},
	}

	// Calculate column width based on rows
	if len(rows) > 0 {
		for _, row := range rows {
			for i, cell := range row {
				if len(cell) > columns[i].Width {
					columns[i].Width = len(cell)
				}
			}
		}
	}

	return columns, rows
}

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
	_       tea.Model = (*model)(nil)
	helpMsg           = "Choose an option from the Framework ID column to use with complytime plan."
)

type model struct {
	table table.Model
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func (m model) View() string {
	return baseStyle.Render(
		m.table.View()) + "\n\n" + helpMsg + "\n"
}
