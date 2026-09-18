package app

import (
	"context"
	"fmt"
	"os"
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
	focusHistory
	focusShortcuts
)

type queryFinished struct {
	result  bigquery.Result
	err     error
	tab     int
	history int
}

type runRecord struct {
	sql     string
	started time.Time
	status  string
	rows    uint64
}

type queryTab struct {
	title         string
	editor        textarea.Model
	results       table.Model
	history       []runRecord
	historyCursor int
}

type model struct {
	client          bigquery.Client
	projects        []project.Project
	active          int
	focus           focus
	tabs            []queryTab
	activeTab       int
	status          string
	width           int
	height          int
	showHelp        bool
	expanded        []bool
	selectedDataset int
}

var (
	accent = lipgloss.Color("#f4b860")
	ink    = lipgloss.Color("#eef0f2")
	muted  = lipgloss.Color("#89919a")
	panel  = lipgloss.Color("#20262b")
	border = lipgloss.Color("#3a444c")
)

func New(client bigquery.Client) *tea.Program {
	return NewWithProjects(client, nil)
}

func initialModel(client bigquery.Client) model {
	return initialModelWithMock(client, os.Getenv("BIGTUI_MOCK_DATA") == "1")
}

func NewWithMock(client bigquery.Client, mock bool) *tea.Program {
	if mock {
		return NewWithProjects(client, project.MockProjects())
	}
	return NewWithProjects(client, nil)
}

func NewWithProjects(client bigquery.Client, projects []project.Project) *tea.Program {
	return tea.NewProgram(initialModelWithProjects(client, projects), tea.WithAltScreen())
}

func initialModelWithMock(client bigquery.Client, mock bool) model {
	projects := []project.Project{}
	if mock || os.Getenv("BIGTUI_MOCK_DATA") == "1" {
		projects = project.MockProjects()
	}
	return initialModelWithProjects(client, projects)
}

func initialModelWithProjects(client bigquery.Client, projects []project.Project) model {
	tab := newQueryTab("Query 1", "")
	tab.editor.Focus()
	return model{
		client: client, projects: projects, focus: focusEditor,
		tabs:     []queryTab{tab},
		status:   "Ready. Ctrl+R runs the query.",
		expanded: make([]bool, len(projects)), selectedDataset: -1,
	}
}

func newQueryTab(title, sql string) queryTab {
	editor := textarea.New()
	editor.Placeholder = "Write SQL..."
	editor.SetValue(sql)
	editor.Prompt = "  "
	editor.CharLimit = 10000
	editor.ShowLineNumbers = true
	editor.SetHeight(7)
	resultTable := table.New(table.WithColumns([]table.Column{{Title: "Result", Width: 22}}), table.WithFocused(false))
	return queryTab{title: title, editor: editor, results: resultTable}
}

func (m model) Init() tea.Cmd { return textarea.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		for index := range m.tabs {
			m.resizeTab(index)
		}
	case queryFinished:
		if msg.tab < 0 || msg.tab >= len(m.tabs) {
			return m, nil
		}
		if msg.history >= 0 && msg.history < len(m.tabs[msg.tab].history) {
			if msg.err != nil {
				m.tabs[msg.tab].history[msg.history].status = "failed"
			} else {
				m.tabs[msg.tab].history[msg.history].status = "done"
				m.tabs[msg.tab].history[msg.history].rows = msg.result.Total
			}
		}
		if msg.err != nil {
			m.status = "Query failed: " + msg.err.Error()
		} else {
			m.status = fmt.Sprintf("Returned %d rows", msg.result.Total)
			m.setResult(msg.tab, msg.result)
		}
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || (msg.String() == "q" && m.focus != focusEditor) {
			return m, tea.Quit
		}
		if msg.String() == "?" && m.focus != focusEditor {
			m.showHelp = !m.showHelp
			return m, nil
		}
		switch msg.String() {
		case "ctrl+n":
			m.addTab()
			return m, nil
		case "ctrl+w":
			m.closeTab()
			return m, nil
		case "alt+left", "ctrl+left", "ctrl+shift+tab":
			m.switchTab(-1)
			return m, nil
		case "alt+right", "ctrl+right", "ctrl+tab":
			m.switchTab(1)
			return m, nil
		}
		if m.showHelp {
			return m, nil
		}
		switch msg.String() {
		case "tab":
			m.focus = (m.focus + 1) % 5
			m.applyFocus()
			return m, nil
		case "shift+tab":
			m.focus = (m.focus + 4) % 5
			m.applyFocus()
			return m, nil
		case "ctrl+r", "ctrl+enter", "ctrl+j":
			if m.focus != focusEditor {
				break
			}
			if len(m.projects) == 0 {
				m.status = "No project selected. Add a project first."
				return m, nil
			}
			m.status = "Running query against " + m.projects[m.active].ID + "..."
			historyIndex := m.recordQuery()
			return m, m.runQuery(historyIndex)
		case "enter":
			if m.focus == focusHistory && len(m.tabs[m.activeTab].history) > 0 {
				record := m.tabs[m.activeTab].history[m.tabs[m.activeTab].historyCursor]
				m.tabs[m.activeTab].editor.SetValue(record.sql)
				m.focus = focusEditor
				m.applyFocus()
				m.status = "Loaded query from run history"
				break
			}
		case "j", "down":
			if m.focus == focusProjects {
				m.moveProjectSelection(1)
			}
			if m.focus == focusHistory && m.tabs[m.activeTab].historyCursor > 0 {
				m.tabs[m.activeTab].historyCursor--
			}
		case "k", "up":
			if m.focus == focusProjects {
				m.moveProjectSelection(-1)
			}
			if m.focus == focusHistory && m.tabs[m.activeTab].historyCursor < len(m.tabs[m.activeTab].history)-1 {
				m.tabs[m.activeTab].historyCursor++
			}
		case "left":
			if m.focus == focusProjects {
				m.collapseProject()
			}
		case "right":
			if m.focus == focusProjects {
				m.expandProject()
			}
		}
	}

	var cmd tea.Cmd
	if m.focus == focusEditor {
		m.tabs[m.activeTab].editor, cmd = m.tabs[m.activeTab].editor.Update(msg)
	} else if m.focus == focusResults {
		m.tabs[m.activeTab].results, cmd = m.tabs[m.activeTab].results.Update(msg)
	}
	return m, cmd
}

func (m *model) applyFocus() {
	for index := range m.tabs {
		m.tabs[index].editor.Blur()
		m.tabs[index].results.Blur()
	}
	if m.focus == focusEditor {
		m.tabs[m.activeTab].editor.Focus()
	}
	m.tabs[m.activeTab].results.SetCursor(0)
}

func (m *model) moveProjectSelection(direction int) {
	if len(m.projects) == 0 {
		return
	}
	if direction > 0 && m.selectedDataset < 0 && m.active < len(m.expanded) && m.expanded[m.active] {
		datasets := m.datasetIndexes(m.active)
		if len(datasets) > 0 {
			m.selectedDataset = datasets[0]
			return
		}
	}
	if m.selectedDataset >= 0 {
		datasets := m.datasetIndexes(m.active)
		position := 0
		for index, resourceIndex := range datasets {
			if resourceIndex == m.selectedDataset {
				position = index
				break
			}
		}
		if direction > 0 && position < len(datasets)-1 {
			m.selectedDataset = datasets[position+1]
			return
		}
		if direction < 0 && position > 0 {
			m.selectedDataset = datasets[position-1]
			return
		}
		if direction < 0 {
			m.selectedDataset = -1
			return
		}
	}
	if direction > 0 && m.active < len(m.projects)-1 {
		m.active++
		m.selectedDataset = -1
	}
	if direction < 0 && m.active > 0 {
		m.active--
		m.selectedDataset = -1
	}
}

func (m model) datasetIndexes(projectIndex int) []int {
	indexes := []int{}
	for index, resource := range m.projects[projectIndex].Resources {
		if resource.Kind == "dataset" {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func (m *model) expandProject() {
	if m.active >= 0 && m.active < len(m.expanded) {
		m.expanded[m.active] = true
	}
}

func (m *model) collapseProject() {
	if m.selectedDataset >= 0 {
		m.selectedDataset = -1
		return
	}
	if m.active >= 0 && m.active < len(m.expanded) {
		m.expanded[m.active] = false
	}
}

func (m *model) addTab() {
	title := fmt.Sprintf("Query %d", len(m.tabs)+1)
	tab := newQueryTab(title, "")
	m.tabs = append(m.tabs, tab)
	m.activeTab = len(m.tabs) - 1
	m.resizeTab(m.activeTab)
	m.applyFocus()
	m.status = "Opened " + title
}

func (m *model) resizeTab(index int) {
	if index < 0 || index >= len(m.tabs) {
		return
	}
	m.tabs[index].editor.SetWidth(max(30, m.width-66))
	m.tabs[index].results.SetWidth(max(30, m.width-66))
	m.tabs[index].results.SetHeight(max(3, m.height-20))
}

func (m *model) closeTab() {
	if len(m.tabs) == 1 {
		m.status = "Cannot close the last tab."
		return
	}
	closed := m.tabs[m.activeTab].title
	m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
	if m.activeTab >= len(m.tabs) {
		m.activeTab = len(m.tabs) - 1
	}
	m.applyFocus()
	m.status = "Closed " + closed
}

func (m *model) switchTab(direction int) {
	if len(m.tabs) < 2 {
		return
	}
	m.activeTab = (m.activeTab + direction + len(m.tabs)) % len(m.tabs)
	m.applyFocus()
	m.status = "Switched to " + m.tabs[m.activeTab].title
}

func (m *model) recordQuery() int {
	tab := &m.tabs[m.activeTab]
	tab.history = append(tab.history, runRecord{sql: tab.editor.Value(), started: time.Now(), status: "running"})
	tab.historyCursor = len(tab.history) - 1
	return tab.historyCursor
}

func (m model) runQuery(historyIndex int) tea.Cmd {
	projectID := m.projects[m.active].ID
	tab := m.activeTab
	sql := m.tabs[tab].editor.Value()
	return func() tea.Msg {
		result, err := m.client.Query(context.Background(), projectID, sql)
		return queryFinished{result: result, err: err, tab: tab, history: historyIndex}
	}
}

func (m *model) setResult(tabIndex int, result bigquery.Result) {
	columns := make([]table.Column, len(result.Columns))
	for i, column := range result.Columns {
		columns[i] = table.Column{Title: column, Width: 18}
	}
	rows := make([]table.Row, len(result.Rows))
	for i, row := range result.Rows {
		rows[i] = table.Row(row.Values)
	}
	m.tabs[tabIndex].results.SetColumns(columns)
	m.tabs[tabIndex].results.SetRows(rows)
}

func (m model) View() string {
	if m.width == 0 {
		return "Starting bigtui..."
	}
	header := lipgloss.NewStyle().Foreground(ink).Bold(true).Render("BIGTUI") + "  " + lipgloss.NewStyle().Foreground(muted).Render("BigQuery workspace")
	tabStrip := m.tabView()
	focusIndicator := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("FOCUS: " + focusLabel(m.focus))
	projectView := lipgloss.JoinVertical(lipgloss.Left, panelTitle("PROJECTS"), m.projectView())
	main := lipgloss.JoinVertical(lipgloss.Left, m.editorView(), m.resultView())
	historyView := lipgloss.JoinVertical(lipgloss.Left, panelTitle("RUN HISTORY"), m.historyView())
	footer := m.shortcutView()
	status := lipgloss.NewStyle().Foreground(accent).Render("● " + m.status)
	workspace := lipgloss.JoinHorizontal(lipgloss.Top, projectView, "  ", main, "  ", historyView)
	view := lipgloss.JoinVertical(lipgloss.Left, header, tabStrip, focusIndicator, "", workspace, "", status, footer)
	if m.showHelp {
		return m.helpView()
	}
	return view
}

func (m model) tabView() string {
	items := make([]string, 0, len(m.tabs))
	for index, tab := range m.tabs {
		label := fmt.Sprintf("%d %s  ×", index+1, tab.title)
		style := lipgloss.NewStyle().Foreground(muted).Padding(0, 1)
		if index == m.activeTab {
			style = style.Foreground(ink).Bold(true).Background(panel).Underline(true)
		}
		items = append(items, style.Render(label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, items...)
}

func (m model) shortcutView() string {
	tabStyle := lipgloss.NewStyle().Foreground(ink).Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(0, 1)
	contextStyle := lipgloss.NewStyle().Foreground(muted).Border(lipgloss.RoundedBorder()).BorderForeground(panel).Padding(0, 1)
	if m.focus == focusShortcuts {
		contextStyle = contextStyle.Foreground(ink).Bold(true).BorderForeground(accent)
	}
	tabControls := tabStyle.Render("TABS  Ctrl+Left/Right switch  ·  Ctrl+N new  ·  Ctrl+W close  ·  Tab focus")
	contextControls := contextStyle.Render(focusShortcutsLabel(m.focus))
	return lipgloss.JoinHorizontal(lipgloss.Top, tabControls, " ", contextControls)
}

func focusShortcutsLabel(current focus) string {
	switch current {
	case focusProjects:
		return "PROJECTS  J/K select"
	case focusEditor:
		return "QUERY EDITOR  Ctrl+R run  ·  Enter newline"
	case focusResults:
		return "RESULTS  Up/Down scroll"
	case focusHistory:
		return "RUN HISTORY  Up/Down select  ·  Enter load"
	case focusShortcuts:
		return "SHORTCUTS  ? help  ·  Q quit"
	default:
		return ""
	}
}

func focusLabel(current focus) string {
	switch current {
	case focusProjects:
		return "PROJECTS"
	case focusEditor:
		return "QUERY EDITOR"
	case focusResults:
		return "RESULTS"
	case focusHistory:
		return "RUN HISTORY"
	case focusShortcuts:
		return "SHORTCUTS"
	default:
		return "UNKNOWN"
	}
}

func (m model) projectView() string {
	if len(m.projects) == 0 {
		content := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(muted).Render("No projects connected."),
			"",
			lipgloss.NewStyle().Foreground(accent).Render("No accessible projects found."),
		)
		return lipgloss.NewStyle().Width(24).Height(max(10, m.height-13)).Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(1).Render(content)
	}
	var lines []string
	for i, item := range m.projects {
		marker := "  "
		if i == m.active && m.selectedDataset < 0 {
			marker = "▸ "
		}
		name := item.Name
		if name == "" {
			name = item.ID
		}
		line := marker + name
		if i == m.active {
			line = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(line)
		}
		lines = append(lines, line)
		if i < len(m.expanded) && m.expanded[i] {
			for resourceIndex, resource := range item.Resources {
				if resource.Kind != "dataset" {
					continue
				}
				marker := "    "
				if i == m.active && resourceIndex == m.selectedDataset {
					marker = "  ▸ "
				}
				line := marker + truncate(resource.Name, 18)
				if i == m.active && resourceIndex == m.selectedDataset {
					line = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(line)
				}
				lines = append(lines, line)
			}
		}
	}
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return lipgloss.NewStyle().Width(24).Height(max(10, m.height-13)).Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(1).Render(content)
}

func (m model) editorView() string {
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("QUERY EDITOR")
	return lipgloss.JoinVertical(lipgloss.Left, title, lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Background(panel).Render(m.tabs[m.activeTab].editor.View()))
}

func (m model) resultView() string {
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("RESULTS")
	return lipgloss.JoinVertical(lipgloss.Left, title, m.tabs[m.activeTab].results.View())
}

func (m model) historyView() string {
	lines := []string{}
	for index := len(m.tabs[m.activeTab].history) - 1; index >= 0; index-- {
		record := m.tabs[m.activeTab].history[index]
		query := strings.Join(strings.Fields(record.sql), " ")
		if len(query) > 25 {
			query = query[:25] + "..."
		}
		marker := "  "
		if index == m.tabs[m.activeTab].historyCursor {
			marker = "▸ "
		}
		line := fmt.Sprintf("%s%d %s [%s]", marker, index+1, query, record.status)
		if index == m.tabs[m.activeTab].historyCursor {
			line = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(line)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render("  No runs yet"))
	}
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return lipgloss.NewStyle().Width(30).Height(max(10, m.height-13)).Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(1).Render(content)
}

func panelTitle(title string) string {
	return lipgloss.NewStyle().Foreground(accent).Bold(true).Render(title)
}

func truncate(value string, width int) string {
	if width < 4 || len(value) <= width {
		return value
	}
	return value[:width-3] + "..."
}

func (m model) helpView() string {
	lines := []string{"KEYMAP", "", "tab / shift+tab   move focus", "ctrl+left/right   switch query tab", "ctrl+n             new query tab", "ctrl+w             close query tab", "j / k              switch project", "ctrl+r             run query", "ctrl+enter         run when supported", "enter              insert newline", "?                  close help", "q                  quit outside editor", "ctrl+c             quit"}
	return lipgloss.NewStyle().Width(50).Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(2).Render(strings.Join(lines, "\n"))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type tickMsg time.Time
