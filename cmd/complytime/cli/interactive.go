package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Example is adapted from the charmbracelet/huh
// Example linked HERE:
// https://github.com/charmbracelet/huh/blob/main/examples/bubbletea/main.go
// This code has been updated to provide functionality for
// updating assessment plans in complytime.

const maxWidth = 80

var (
	red = lipgloss.AdaptiveColor{Light: "#FE5F86", Dark: "#FE5F86"}
	//indigo = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
	//green  = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
	bluey  = lipgloss.AdaptiveColor{Light: "#155abb", Dark: "#006de1"}
	bluey2 = lipgloss.AdaptiveColor{Light: "#0065a9", Dark: "#0098ff"}
	//complyblue = lipgloss.AdaptiveColor{Light: "#5755ff", Dark: "#5755ff"}
	complyblue = lipgloss.AdaptiveColor{Light: "#4b4d4a", Dark: "#4b4d4a"}
)

type Styles struct {
	Base,
	HeaderText,
	Status,
	StatusHeader,
	Highlight,
	ErrorHeaderText,
	Help lipgloss.Style
}

func NewStyles(lg *lipgloss.Renderer) *Styles {
	s := Styles{}
	s.Base = lg.NewStyle().
		Padding(1, 4, 0, 1)
	s.HeaderText = lg.NewStyle().
		Foreground(bluey).
		Bold(true).
		Padding(0, 1, 0, 2)
	s.Status = lg.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bluey).
		PaddingLeft(1).
		MarginTop(1)
	s.StatusHeader = lg.NewStyle().
		Foreground(bluey2).
		Bold(true)
	s.Highlight = lg.NewStyle().
		Background(complyblue).Bold(true).
		Foreground(lipgloss.Color("212"))
	s.ErrorHeaderText = s.HeaderText.
		Foreground(red)
	s.Help = lg.NewStyle().
		Foreground(lipgloss.Color("240"))
	//Foreground(lipgloss.Color("#5755ff"))
	return &s
}

type state int

const (
	statusNormal state = iota
	stateDone
)

type Model struct {
	state  state
	lg     *lipgloss.Renderer
	styles *Styles
	form   *huh.Form
	width  int
}

func NewModel() Model {
	m := Model{width: maxWidth}
	m.lg = lipgloss.DefaultRenderer()
	m.styles = NewStyles(m.lg)

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("component").
				Options(huh.NewOptions("Controls", "Rules", "Parameters")...).
				//Options(huh.NewOptions("Controls", "Rules", "Parameters")...).
				Title("Choose component(s) to update.").
				Description("This will choose the component to edit."),
			huh.NewSelect[string]().
				Key("revisions").
				Options(huh.NewOptions("exclude", "update", "include")...).
				Title("Choose your revisions for each component.").
				Description("Change to be made for each component."),
			huh.NewConfirm().
				Key("done").
				Title("All done?").
				Validate(func(v bool) error {
					if !v {
						return fmt.Errorf("Complete the final updates before submission. ")
					}
					return nil
				}).
				Affirmative("I'm done.").
				Negative("Not yet."),
		),
	).
		WithWidth(55).
		WithShowHelp(false).
		WithShowErrors(false)
	return m
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(msg.Width, maxWidth) - m.styles.Base.GetHorizontalFrameSize()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Interrupt
		case "esc", "q":
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd

	// Process the form
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		cmds = append(cmds, cmd)
	}

	if m.form.State == huh.StateCompleted {
		// Quit when the form is done.
		cmds = append(cmds, tea.Quit)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	s := m.styles

	switch m.form.State {
	case huh.StateCompleted:
		component, role := m.getRole()
		component = s.Highlight.Render(component)
		var b strings.Builder
		fmt.Fprintf(&b, "You successfully edited your assessment plan\n%s!\n\n", component)
		fmt.Fprintf(&b, "Your updates made:\n\n%s\n\nReference your assessment-plan.json.", role)
		return s.Status.Margin(0, 1).Padding(1, 2).Width(48).Render(b.String()) + "\n\n"
	default:

		var component string
		if m.form.GetString("component") != "" {
			component = s.Base.Render(component)
			component = "Component Selected: " + m.form.GetString("component")
		}

		// Form (left side)
		v := strings.TrimSuffix(m.form.View(), "\n\n")
		form := m.lg.NewStyle().Margin(1, 0).Render(v)
		// form = m.lg.NewStyle().Inherit(s.Base).Render(form)
		//form := m.lg.NewStyle().Background(indigo).Margin(1, 0).Render(v)

		// Status (right side)
		var status string
		{
			var (
				buildInfo      = "frameworkid defaults"
				role           string
				jobDescription string
				revisions      string
			)

			if m.form.GetString("revisions") != "" {
				revisions = "Revisions: " + m.form.GetString("revisions")
				role, jobDescription = m.getRole()
				role = "\n\n" + s.StatusHeader.Render("Assessment Plan Diff") + "\n" + role
				jobDescription = "\n\n" + s.StatusHeader.Render("Changes made:") + "\n" + jobDescription
			}
			if m.form.GetString("component") != "" {
				buildInfo = fmt.Sprintf("%s\n%s\n", component, revisions)
			}
			const statusWidth = 35
			statusMarginLeft := m.width - statusWidth - lipgloss.Width(form) - s.Status.GetMarginRight()
			status = s.Status.
				Height(lipgloss.Height(form)).
				Width(statusWidth).
				MarginLeft(statusMarginLeft).
				Render(s.StatusHeader.Render("Staged Plan Updates") + "\n" +
					buildInfo +
					role +
					jobDescription)
		}

		errors := m.form.Errors()
		header := m.appBoundaryView("Assessment Plan Editing")
		if len(errors) > 0 {
			header = m.appErrorBoundaryView(m.errorView())
		}
		body := lipgloss.JoinHorizontal(lipgloss.Left, form, status)

		footer := m.appBoundaryView(m.form.Help().ShortHelpView(m.form.KeyBinds()))
		if len(errors) > 0 {
			footer = m.appErrorBoundaryView("")
		}

		return s.Base.Render(header + "\n" + body + "\n\n" + footer)
	}
}

func (m Model) errorView() string {
	var s string
	for _, err := range m.form.Errors() {
		s += err.Error()
	}
	return s
}

func (m Model) appBoundaryView(text string) string {
	return lipgloss.PlaceHorizontal(
		m.width,
		lipgloss.Left,
		m.styles.HeaderText.Render(text),
		lipgloss.WithWhitespaceChars("/"),
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#5e5e5e")),
		lipgloss.WithWhitespaceForeground(bluey2),
	)
}

func (m Model) appErrorBoundaryView(text string) string {
	return lipgloss.PlaceHorizontal(
		m.width,
		lipgloss.Left,
		m.styles.ErrorHeaderText.Render(text),
		lipgloss.WithWhitespaceChars("/"),
		lipgloss.WithWhitespaceChars("/"),
		lipgloss.WithWhitespaceForeground(red),
	)
}

func (m Model) getRole() (string, string) {
	revisions := m.form.GetString("revisions")
	switch m.form.GetString("component") {
	case "Controls":
		switch revisions {
		case "exclude":
			return "Excluded controls from assessment-plan.json", "Excludes controls from updated assessment-plan.json."
		case "update":
			return "Update Controls", "Updates status of controls from 'assess/assessed' to 'waived' in plan."
		case "include":
			return "Includes Defaults", "Includes all default controls from the framework id passed to complytime."
		default:
			return "No changes", "The assessment-plan.json will populate with the defaults from the frameworkid."
		}
	case "Rules":
		switch revisions {
		case "exclude":
			return "Excluded rules from assessment-plan.json", "Excludes rules from updated assessment-plan.json."
		case "update":
			return "Update Controls", "Updates status of controls from 'assess/assessed' to 'waived' in plan."
		case "include":
			return "Includes Defaults", "Includes all default rules from the framework id passed to complytime."
		default:
			return "No changes", "The assessment-plan.json will populate with the defaults from the frameworkid."
		}
	case "Parameters":
		switch revisions {
		case "exclude":
			return "Not applicable.", "You must keep the parameters, only update status."
		case "update":
			return "Update Parameters", "Updates status of parameters from 'assess/assessed' to 'waived' in plan."
		case "include":
			return "Includes Defaults", "Includes all default parameters from the framework id passed to complytime."
		default:
			return "No changes", "The assessment-plan.json will populate with the defaults from the frameworkid."
		}

	default:
		return "", ""
	}
}
