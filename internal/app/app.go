package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xjrr/bigtui/internal/bigquery"
	"github.com/xjrr/bigtui/internal/project"
)

type focus int

const (
	focusProjects focus = iota
	focusEditor
	focusResults
	focusShortcuts
)

type queryFinished struct {
	result bigquery.Result
	err    error
}

type model struct {
	client   bigquery.Client
	projects []project.Project
	active   int
	focus    focus
	editor   textarea.Model
	results  table.Model
	status   string
	width    int
	height   int
	showHelp bool
}

var (
	accent = lipgloss.Color("#f4b860")
	ink    = lipgloss.Color("#eef0f2")
	muted  = lipgloss.Color("#89919a")
	panel  = lipgloss.Color("#20262b")
	border = lipgloss.Color("#3a444c")
)

func New(client bigquery.Client) *tea.Program {
	return tea.NewProgram(initialModel(client), tea.WithAltScreen())
}

func initialModel(client bigquery.Client) model {
	editor := textarea.New()
	editor.Placeholder = "Write SQL..."
	editor.SetValue("SELECT project_id, COUNT(*) AS rows\nFROM `demo-analytics.region-us.INFORMATION_SCHEMA.TABLES`\nGROUP BY project_id\nORDER BY rows DESC")
	editor.Prompt = "  "
	editor.CharLimit = 10000
	editor.ShowLineNumbers = true
	editor.SetHeight(7)
	editor.Focus()

	resultTable := table.New(table.WithColumns([]table.Column{{Title: "Result", Width: 22}}), table.WithFocused(false))
	return model{
		client: client, projects: project.Defaults, focus: focusEditor,
		editor: editor, results: resultTable,
		status: "Ready. Ctrl+Enter runs the query.",
	}
}

func (m model) Init() tea.Cmd { return textarea.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.editor.SetWidth(max(30, msg.Width-34))
		m.results.SetWidth(max(30, msg.Width-34))
		m.results.SetHeight(max(3, msg.Height-18))
	case queryFinished:
		if msg.err != nil {
			m.status = "Query failed: " + msg.err.Error()
		} else {
			m.status = fmt.Sprintf("Returned %d rows", msg.result.Total)
			m.setResult(msg.result)
		}
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "?" {
			m.showHelp = !m.showHelp
			return m, nil
		}
		if m.showHelp {
			return m, nil
		}
		switch msg.String() {
		case "tab":
			m.focus = (m.focus + 1) % 4
			m.applyFocus()
			return m, nil
		case "shift+tab":
			m.focus = (m.focus + 3) % 4
			m.applyFocus()
			return m, nil
		case "ctrl+enter":
			m.status = "Running query against " + m.projects[m.active].ID + "..."
			return m, m.runQuery()
		case "a":
			m.projects = append(m.projects, project.Project{ID: "new-project", Location: "US"})
			m.active = len(m.projects) - 1
			m.status = "Added project placeholder; edit project configuration next."
			return m, nil
		case "j", "down":
			if m.focus == focusProjects && m.active < len(m.projects)-1 {
				m.active++
			}
		case "k", "up":
			if m.focus == focusProjects && m.active > 0 {
				m.active--
			}
		}
	}

	var cmd tea.Cmd
	if m.focus == focusEditor {
		m.editor, cmd = m.editor.Update(msg)
	} else if m.focus == focusResults {
		m.results, cmd = m.results.Update(msg)
	}
	return m, cmd
}

func (m *model) applyFocus() {
	m.editor.Blur()
	m.results.Blur()
	if m.focus == focusEditor {
		m.editor.Focus()
	}
	m.results.SetCursor(0)
}

func (m model) runQuery() tea.Cmd {
	projectID := m.projects[m.active].ID
	sql := m.editor.Value()
	return func() tea.Msg {
		result, err := m.client.Query(context.Background(), projectID, sql)
		return queryFinished{result: result, err: err}
	}
}

func (m *model) setResult(result bigquery.Result) {
	columns := make([]table.Column, len(result.Columns))
	for i, column := range result.Columns {
		columns[i] = table.Column{Title: column, Width: 18}
	}
	rows := make([]table.Row, len(result.Rows))
	for i, row := range result.Rows {
		rows[i] = table.Row(row.Values)
	}
	m.results.SetColumns(columns)
	m.results.SetRows(rows)
}

func (m model) View() string {
	if m.width == 0 {
		return "Starting bigtui..."
	}
	header := lipgloss.NewStyle().Foreground(ink).Bold(true).Render("BIGTUI") + "  " + lipgloss.NewStyle().Foreground(muted).Render("BigQuery workspace")
	focusIndicator := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("FOCUS: " + focusLabel(m.focus))
	projectView := m.projectView()
	main := lipgloss.JoinVertical(lipgloss.Left, m.editorView(), m.resultView())
	footerStyle := lipgloss.NewStyle().Foreground(muted).Border(lipgloss.RoundedBorder()).BorderForeground(panel).Padding(0, 1)
	if m.focus == focusShortcuts {
		footerStyle = footerStyle.Foreground(ink).Bold(true).BorderForeground(accent)
	}
	footer := footerStyle.Render("tab focus  •  ctrl+enter run  •  a add project  •  ? help  •  q quit")
	status := lipgloss.NewStyle().Foreground(accent).Render("● " + m.status)
	view := lipgloss.JoinVertical(lipgloss.Left, header, focusIndicator, "", lipgloss.JoinHorizontal(lipgloss.Top, projectView, "  ", main), "", status, footer)
	if m.showHelp {
		return m.helpView()
	}
	return view
}

func focusLabel(current focus) string {
	switch current {
	case focusProjects:
		return "PROJECTS"
	case focusEditor:
		return "QUERY EDITOR"
	case focusResults:
		return "RESULTS"
	case focusShortcuts:
		return "SHORTCUTS"
	default:
		return "UNKNOWN"
	}
}

func (m model) projectView() string {
	var lines []string
	for i, item := range m.projects {
		marker := "  "
		if i == m.active {
			marker = "▸ "
		}
		line := marker + item.ID
		if i == m.active {
			line = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(line)
		}
		lines = append(lines, line)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, append([]string{lipgloss.NewStyle().Foreground(muted).Bold(true).Render("PROJECTS")}, lines...)...)
	return lipgloss.NewStyle().Width(24).Height(max(10, m.height-6)).Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(1).Render(content)
}

func (m model) editorView() string {
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("QUERY EDITOR")
	return lipgloss.JoinVertical(lipgloss.Left, title, lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Background(panel).Render(m.editor.View()))
}

func (m model) resultView() string {
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("RESULTS")
	return lipgloss.JoinVertical(lipgloss.Left, title, m.results.View())
}

func (m model) helpView() string {
	lines := []string{"KEYMAP", "", "tab / shift+tab   move focus", "j / k              switch project", "ctrl+enter         execute query", "a                  add project", "?                  close help", "ctrl+c             quit"}
	return lipgloss.NewStyle().Width(50).Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(2).Render(strings.Join(lines, "\n"))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type tickMsg time.Time
