package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
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

type queryAnalyzed struct{ analysis bigquery.Analysis }

type projectsLoaded struct {
	projects []project.Project
	err      error
}

type datasetsLoaded struct {
	projectIndex int
	resources    []project.Resource
	err          error
}

type tablesLoaded struct {
	projectIndex int
	datasetIndex int
	resources    []project.Resource
	err          error
}

type resourceLoaded struct {
	projectIndex int
	datasetIndex int
	childIndex   int
	resource     project.Resource
	err          error
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
	result        bigquery.Result
	resultRow     int
	resultColumn  int
	resultOffset  int
}

type searchResult struct {
	projectIndex int
	datasetIndex int
	childIndex   int
	kind         string
	name         string
	project      string
	projectID    string
	dataset      string
}

type projectRow struct {
	text     string
	selected bool
}

type model struct {
	client          bigquery.Client
	loader          project.CatalogLoader
	projects        []project.Project
	active          int
	focus           focus
	tabs            []queryTab
	activeTab       int
	status          string
	validation      string
	width           int
	height          int
	showHelp        bool
	expanded        []bool
	selectedDataset int
	selectedChild   int
	expandedDataset map[string]bool
	showInfo        bool
	infoScroll      int
	searchOpen      bool
	searchInput     textinput.Model
	searchResults   []searchResult
	searchCursor    int
	resourceLoading bool
	projectLoading  map[int]bool
	datasetLoading  map[string]bool
	datasetsLoaded  map[int]bool
	tablesLoaded    map[string]bool
	projectScroll   int
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
	return NewWithProjectsAndLoader(client, projects, nil)
}

func NewWithProjectsAndLoader(client bigquery.Client, projects []project.Project, loader project.CatalogLoader) *tea.Program {
	return tea.NewProgram(initialModelWithProjectsAndLoader(client, projects, loader), tea.WithAltScreen())
}

func NewWithLoader(client bigquery.Client, loader project.CatalogLoader) *tea.Program {
	return tea.NewProgram(initialModelWithProjectsAndLoader(client, nil, loader), tea.WithAltScreen())
}

func initialModelWithMock(client bigquery.Client, mock bool) model {
	projects := []project.Project{}
	if mock || os.Getenv("BIGTUI_MOCK_DATA") == "1" {
		projects = project.MockProjects()
	}
	return initialModelWithProjects(client, projects)
}

func initialModelWithProjects(client bigquery.Client, projects []project.Project) model {
	return initialModelWithProjectsAndLoader(client, projects, nil)
}

func initialModelWithProjectsAndLoader(client bigquery.Client, projects []project.Project, loader project.CatalogLoader) model {
	tab := newQueryTab("Query 1", "")
	tab.editor.Focus()
	searchInput := textinput.New()
	searchInput.Placeholder = "Search projects, datasets, tables, and views..."
	searchInput.CharLimit = 120
	status := "Ready. Ctrl+R runs the query."
	if loader != nil && len(projects) == 0 {
		status = "Loading projects..."
	}
	return model{
		client: client, loader: loader, projects: projects, focus: focusEditor,
		tabs:     []queryTab{tab},
		status:   status,
		expanded: make([]bool, len(projects)), selectedDataset: -1, selectedChild: -1, expandedDataset: map[string]bool{},
		projectLoading: map[int]bool{}, datasetLoading: map[string]bool{}, datasetsLoaded: map[int]bool{}, tablesLoaded: map[string]bool{},
		searchInput: searchInput,
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

func (m model) Init() tea.Cmd {
	if m.loader != nil && len(m.projects) == 0 {
		return tea.Batch(textarea.Blink, m.loadProjects())
	}
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		for index := range m.tabs {
			m.resizeTab(index)
		}
		m.updateProjectScroll()
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
	case queryAnalyzed:
		if msg.analysis.Err != nil {
			m.validation = "0 Invalid · " + truncate(formatValidationError(msg.analysis.Err), 70)
		} else if msg.analysis.Valid {
			m.validation = fmt.Sprintf("1 Valid · %s processed", formatBytes(msg.analysis.BytesProcessed))
		}
	case projectsLoaded:
		if msg.err != nil {
			m.status = "Project loading failed: " + msg.err.Error()
			return m, nil
		}
		m.projects = msg.projects
		m.expanded = make([]bool, len(msg.projects))
		m.selectedDataset = -1
		m.selectedChild = -1
		m.projectLoading = map[int]bool{}
		m.datasetLoading = map[string]bool{}
		m.datasetsLoaded = map[int]bool{}
		m.tablesLoaded = map[string]bool{}
		m.projectScroll = 0
		m.updateProjectScroll()
		m.status = fmt.Sprintf("Ready. Loaded %d projects.", len(msg.projects))
	case datasetsLoaded:
		m.projectLoading[msg.projectIndex] = false
		if msg.err != nil {
			m.status = "Dataset loading failed: " + msg.err.Error()
			return m, nil
		}
		if msg.projectIndex >= 0 && msg.projectIndex < len(m.projects) {
			m.projects[msg.projectIndex].Resources = msg.resources
			m.datasetsLoaded[msg.projectIndex] = true
			m.updateProjectScroll()
			m.status = fmt.Sprintf("Loaded %d datasets.", len(msg.resources))
		}
	case tablesLoaded:
		key := m.datasetKey(msg.projectIndex, msg.datasetIndex)
		m.datasetLoading[key] = false
		if msg.err != nil {
			m.status = "Table loading failed: " + msg.err.Error()
			return m, nil
		}
		if msg.projectIndex >= 0 && msg.projectIndex < len(m.projects) && msg.datasetIndex >= 0 && msg.datasetIndex < len(m.projects[msg.projectIndex].Resources) {
			m.projects[msg.projectIndex].Resources[msg.datasetIndex].Children = msg.resources
			m.projects[msg.projectIndex].Resources[msg.datasetIndex].ChildrenLoaded = true
			m.tablesLoaded[key] = true
			m.updateProjectScroll()
			m.status = fmt.Sprintf("Loaded %d tables/views.", len(msg.resources))
		}
	case resourceLoaded:
		m.resourceLoading = false
		if msg.err != nil {
			m.status = "Resource details failed: " + msg.err.Error()
			return m, nil
		}
		if msg.projectIndex < len(m.projects) && msg.datasetIndex >= 0 && msg.datasetIndex < len(m.projects[msg.projectIndex].Resources) {
			dataset := &m.projects[msg.projectIndex].Resources[msg.datasetIndex]
			if msg.childIndex >= 0 && msg.childIndex < len(dataset.Children) {
				dataset.Children[msg.childIndex] = msg.resource
				m.status = "Loaded details for " + msg.resource.Name
			}
		}
	case tea.KeyMsg:
		if m.searchOpen {
			switch msg.String() {
			case "esc", "ctrl+s":
				m.searchOpen = false
				m.searchInput.Blur()
				return m, nil
			case "enter":
				m.selectSearchResult()
				return m, nil
			case "up", "ctrl+k":
				if m.searchCursor > 0 {
					m.searchCursor--
				}
				return m, nil
			case "down", "ctrl+j":
				if m.searchCursor < len(m.searchResults)-1 {
					m.searchCursor++
				}
				return m, nil
			}
			var searchCmd tea.Cmd
			m.searchInput, searchCmd = m.searchInput.Update(msg)
			m.refreshSearch()
			return m, searchCmd
		}
		if m.showInfo {
			switch msg.String() {
			case "esc", "enter", "q":
				m.showInfo = false
				m.infoScroll = 0
				return m, nil
			case "up", "k":
				if m.infoScroll > 0 {
					m.infoScroll--
				}
				return m, nil
			case "down", "j":
				m.infoScroll++
				return m, nil
			case "pgup":
				m.infoScroll -= max(1, m.height/2)
				if m.infoScroll < 0 {
					m.infoScroll = 0
				}
				return m, nil
			case "pgdown":
				m.infoScroll += max(1, m.height/2)
				return m, nil
			}
			return m, nil
		}
		if msg.String() == "ctrl+c" || (msg.String() == "q" && m.focus != focusEditor) {
			return m, tea.Quit
		}
		if msg.String() == "?" && m.focus != focusEditor {
			m.showHelp = !m.showHelp
			return m, nil
		}
		switch msg.String() {
		case "ctrl+s":
			m.searchOpen = true
			m.searchInput.SetValue("")
			m.searchInput.Focus()
			m.refreshSearch()
			return m, nil
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
			if m.focus == focusProjects && len(m.projects) > 0 {
				m.showInfo = true
				m.infoScroll = 0
				if m.selectedDataset >= 0 && m.selectedChild >= 0 {
					child := m.projects[m.active].Resources[m.selectedDataset].Children[m.selectedChild]
					if !child.DetailsLoaded && m.loader != nil && !m.resourceLoading {
						m.resourceLoading = true
						m.status = "Loading details for " + child.Name + "..."
						return m, m.loadResource(m.active, m.selectedDataset, m.selectedChild)
					}
				}
				return m, nil
			}
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
			if m.focus == focusResults {
				m.moveResultRow(1)
			}
		case "k", "up":
			if m.focus == focusProjects {
				m.moveProjectSelection(-1)
			}
			if m.focus == focusHistory && m.tabs[m.activeTab].historyCursor < len(m.tabs[m.activeTab].history)-1 {
				m.tabs[m.activeTab].historyCursor++
			}
			if m.focus == focusResults {
				m.moveResultRow(-1)
			}
		case "left", "h":
			if m.focus == focusProjects {
				m.collapseProject()
			}
			if m.focus == focusResults {
				m.moveResultColumn(-1)
			}
		case "right", "l":
			if m.focus == focusProjects {
				return m, m.expandProject()
			}
			if m.focus == focusResults {
				m.moveResultColumn(1)
			}
		}
	}

	var cmd tea.Cmd
	if m.focus == focusEditor {
		m.tabs[m.activeTab].editor, cmd = m.tabs[m.activeTab].editor.Update(msg)
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Batch(cmd, m.analyzeQuery())
		}
	} else if m.focus == focusResults {
		m.tabs[m.activeTab].results, cmd = m.tabs[m.activeTab].results.Update(msg)
	}
	return m, cmd
}

func (m model) analyzeQuery() tea.Cmd {
	if len(m.projects) == 0 {
		return nil
	}
	analyzer, ok := m.client.(bigquery.Analyzer)
	if !ok {
		return nil
	}
	projectID := m.projects[m.active].ID
	sql := m.tabs[m.activeTab].editor.Value()
	return func() tea.Msg { return queryAnalyzed{analysis: analyzer.Analyze(context.Background(), projectID, sql)} }
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

func (m *model) refreshSearch() {
	term := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	m.searchResults = nil
	for projectIndex, item := range m.projects {
		projectName := item.Name
		if projectName == "" {
			projectName = item.ID
		}
		if term == "" || strings.Contains(strings.ToLower(projectName), term) || strings.Contains(strings.ToLower(item.ID), term) {
			m.searchResults = append(m.searchResults, searchResult{projectIndex: projectIndex, datasetIndex: -1, childIndex: -1, kind: "project", name: projectName, project: projectName, projectID: item.ID})
		}
		for datasetIndex, resource := range item.Resources {
			if resource.Kind != "dataset" {
				continue
			}
			if matchesSearch(term, resource.Name, projectName, item.ID) {
				m.searchResults = append(m.searchResults, searchResult{projectIndex: projectIndex, datasetIndex: datasetIndex, childIndex: -1, kind: "dataset", name: resource.Name, project: projectName, projectID: item.ID})
			}
			for childIndex, child := range resource.Children {
				if matchesSearch(term, child.Name, projectName, item.ID, resource.Name) {
					m.searchResults = append(m.searchResults, searchResult{projectIndex: projectIndex, datasetIndex: datasetIndex, childIndex: childIndex, kind: child.Kind, name: child.Name, project: projectName, projectID: item.ID, dataset: resource.Name})
				}
			}
		}
	}
	if m.searchCursor >= len(m.searchResults) {
		m.searchCursor = max(0, len(m.searchResults)-1)
	}
}

func matchesSearch(term, name string, context ...string) bool {
	if term == "" {
		return true
	}
	values := append([]string{name}, context...)
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), term) {
			return true
		}
	}
	return false
}

func (m *model) selectSearchResult() {
	if len(m.searchResults) == 0 {
		return
	}
	result := m.searchResults[m.searchCursor]
	m.active = result.projectIndex
	m.selectedDataset = result.datasetIndex
	m.selectedChild = result.childIndex
	if result.projectIndex < len(m.expanded) {
		m.expanded[result.projectIndex] = result.datasetIndex >= 0
	}
	if result.datasetIndex >= 0 {
		m.expandedDataset[m.datasetKey(result.projectIndex, result.datasetIndex)] = result.childIndex >= 0
	}
	m.searchOpen = false
	m.searchInput.Blur()
	m.focus = focusProjects
	m.applyFocus()
	m.status = "Selected " + result.name
}

func (m *model) moveProjectSelection(direction int) {
	defer m.updateProjectScroll()
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
		if m.selectedChild >= 0 {
			children := m.projects[m.active].Resources[m.selectedDataset].Children
			if direction > 0 && m.selectedChild < len(children)-1 {
				m.selectedChild++
				return
			}
			if direction < 0 && m.selectedChild > 0 {
				m.selectedChild--
				return
			}
			if direction < 0 {
				m.selectedChild = -1
				return
			}
		}
		if direction > 0 && m.datasetExpanded(m.active, m.selectedDataset) {
			children := m.projects[m.active].Resources[m.selectedDataset].Children
			if len(children) > 0 {
				m.selectedChild = 0
				return
			}
		}
		position := 0
		for index, resourceIndex := range datasets {
			if resourceIndex == m.selectedDataset {
				position = index
				break
			}
		}
		if direction > 0 && position < len(datasets)-1 {
			m.selectedDataset = datasets[position+1]
			m.selectedChild = -1
			return
		}
		if direction < 0 && position > 0 {
			m.selectedDataset = datasets[position-1]
			m.selectedChild = -1
			return
		}
		if direction < 0 {
			m.selectedDataset = -1
			m.selectedChild = -1
			return
		}
	}
	if direction > 0 && m.active < len(m.projects)-1 {
		m.active++
		m.selectedDataset = -1
		m.selectedChild = -1
	}
	if direction < 0 && m.active > 0 {
		m.active--
		m.selectedDataset = -1
		m.selectedChild = -1
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

func (m model) datasetKey(projectIndex, resourceIndex int) string {
	return fmt.Sprintf("%d:%d", projectIndex, resourceIndex)
}
func (m model) datasetExpanded(projectIndex, resourceIndex int) bool {
	return m.expandedDataset[m.datasetKey(projectIndex, resourceIndex)]
}

func (m *model) expandProject() tea.Cmd {
	defer m.updateProjectScroll()
	if m.selectedDataset >= 0 {
		m.expandedDataset[m.datasetKey(m.active, m.selectedDataset)] = true
		if m.loader != nil && !m.tablesLoaded[m.datasetKey(m.active, m.selectedDataset)] && !m.datasetLoading[m.datasetKey(m.active, m.selectedDataset)] {
			m.datasetLoading[m.datasetKey(m.active, m.selectedDataset)] = true
			m.status = "Loading tables and views..."
			return m.loadTables(m.active, m.selectedDataset)
		}
		return nil
	}
	if m.active >= 0 && m.active < len(m.expanded) {
		m.expanded[m.active] = true
		if m.loader != nil && !m.datasetsLoaded[m.active] && !m.projectLoading[m.active] {
			m.projectLoading[m.active] = true
			m.status = "Loading datasets..."
			return m.loadDatasets(m.active)
		}
	}
	return nil
}

func (m *model) collapseProject() {
	defer m.updateProjectScroll()
	if m.selectedChild >= 0 {
		m.selectedChild = -1
		return
	}
	if m.selectedDataset >= 0 {
		if m.datasetExpanded(m.active, m.selectedDataset) {
			delete(m.expandedDataset, m.datasetKey(m.active, m.selectedDataset))
			return
		}
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
	reserve := 14
	if m.width < 110 {
		reserve = 19
	}
	if m.width < 90 {
		reserve = 29
	}
	contentWidth := max(10, m.width-m.projectPanelWidth()-m.historyPanelWidth()-reserve)
	m.tabs[index].editor.SetWidth(contentWidth)
	m.tabs[index].results.SetWidth(contentWidth)
	m.resizeResultColumns(index)
	m.tabs[index].results.SetHeight(max(3, m.height-23))
}

func (m *model) resizeResultColumns(tabIndex int) {
	columns := m.tabs[tabIndex].results.Columns()
	if len(columns) == 0 {
		return
	}
	columnWidth := max(6, (m.tabs[tabIndex].results.Width()-2*max(0, len(columns)-1))/len(columns))
	for index := range columns {
		columns[index].Width = columnWidth
	}
	m.tabs[tabIndex].results.SetColumns(columns)
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

func (m model) loadResource(projectIndex, datasetIndex, childIndex int) tea.Cmd {
	loader := m.loader
	projectID := m.projects[projectIndex].ID
	datasetID := m.projects[projectIndex].Resources[datasetIndex].Name
	tableID := m.projects[projectIndex].Resources[datasetIndex].Children[childIndex].Name
	return func() tea.Msg {
		resource, err := loader.LoadResource(context.Background(), projectID, datasetID, tableID)
		return resourceLoaded{projectIndex: projectIndex, datasetIndex: datasetIndex, childIndex: childIndex, resource: resource, err: err}
	}
}

func (m model) loadProjects() tea.Cmd {
	loader := m.loader
	return func() tea.Msg {
		projects, err := loader.Load(context.Background())
		return projectsLoaded{projects: projects, err: err}
	}
}

func (m model) loadDatasets(projectIndex int) tea.Cmd {
	loader := m.loader
	projectID := m.projects[projectIndex].ID
	return func() tea.Msg {
		resources, err := loader.LoadDatasets(context.Background(), projectID)
		return datasetsLoaded{projectIndex: projectIndex, resources: resources, err: err}
	}
}

func (m model) loadTables(projectIndex, datasetIndex int) tea.Cmd {
	loader := m.loader
	projectID := m.projects[projectIndex].ID
	datasetID := m.projects[projectIndex].Resources[datasetIndex].Name
	return func() tea.Msg {
		resources, err := loader.LoadTables(context.Background(), projectID, datasetID)
		return tablesLoaded{projectIndex: projectIndex, datasetIndex: datasetIndex, resources: resources, err: err}
	}
}

func (m *model) setResult(tabIndex int, result bigquery.Result) {
	m.tabs[tabIndex].result = result
	columns := make([]table.Column, len(result.Columns))
	columnWidth := max(6, (m.tabs[tabIndex].results.Width()-2*max(0, len(result.Columns)-1))/max(1, len(result.Columns)))
	for i, column := range result.Columns {
		columns[i] = table.Column{Title: truncate(column, columnWidth), Width: columnWidth}
	}
	rows := make([]table.Row, len(result.Rows))
	for i, row := range result.Rows {
		rows[i] = table.Row(row.Values)
	}
	m.tabs[tabIndex].results.SetColumns(columns)
	m.tabs[tabIndex].results.SetRows(rows)
	m.resizeResultColumns(tabIndex)
}

func (m *model) moveResultRow(direction int) {
	tab := &m.tabs[m.activeTab]
	if len(tab.result.Rows) == 0 {
		return
	}
	tab.resultRow += direction
	if tab.resultRow < 0 {
		tab.resultRow = 0
	}
	if tab.resultRow >= len(tab.result.Rows) {
		tab.resultRow = len(tab.result.Rows) - 1
	}
}

func (m *model) moveResultColumn(direction int) {
	tab := &m.tabs[m.activeTab]
	if len(tab.result.Columns) == 0 {
		return
	}
	tab.resultColumn += direction
	if tab.resultColumn < 0 {
		tab.resultColumn = 0
	}
	if tab.resultColumn >= len(tab.result.Columns) {
		tab.resultColumn = len(tab.result.Columns) - 1
	}
	if tab.resultColumn < tab.resultOffset {
		tab.resultOffset = tab.resultColumn
	}
	if tab.resultColumn >= tab.resultOffset+1 {
		tab.resultOffset = tab.resultColumn
	}
}

func (m model) View() string {
	if m.width == 0 {
		return "Starting bigtui..."
	}
	if m.showInfo {
		return m.infoView()
	}
	if m.searchOpen {
		return m.searchView()
	}
	header := lipgloss.NewStyle().Foreground(ink).Bold(true).Render("BIGTUI") + "  " + lipgloss.NewStyle().Foreground(muted).Render("BigQuery workspace")
	tabStrip := m.tabView()
	focusIndicator := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("FOCUS: " + focusLabel(m.focus))
	projectView := lipgloss.JoinVertical(lipgloss.Left, panelTitle("PROJECTS"), m.projectView(), m.selectedResourceView())
	main := lipgloss.JoinVertical(lipgloss.Left, m.editorView(), m.resultView())
	historyView := lipgloss.JoinVertical(lipgloss.Left, panelTitle("RUN HISTORY"), m.historyView())
	footer := m.shortcutView()
	status := lipgloss.NewStyle().Foreground(accent).Render("● " + m.status)
	validation := lipgloss.NewStyle().Foreground(muted).Render(m.validation)
	workspace := lipgloss.JoinHorizontal(lipgloss.Top, projectView, "  ", main, "  ", historyView)
	view := lipgloss.JoinVertical(lipgloss.Left, header, tabStrip, focusIndicator, "", workspace, "", status, validation, footer)
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
	tabText := "TABS  Ctrl+Left/Right switch  ·  Ctrl+N new  ·  Ctrl+W close  ·  Tab focus"
	if m.width < 140 {
		tabText = "TABS  Ctrl+Left/Right  ·  Ctrl+N  ·  Ctrl+W"
	}
	if m.width < 100 {
		tabText = "TABS  Ctrl+Left/Right"
	}
	tabControls := tabStyle.Render(tabText)
	contextLabel := focusShortcutsLabel(m.focus)
	if m.focus == focusProjects && m.width < 140 {
		contextLabel = "PROJECTS  Up/Down  ·  Left/Right  ·  Enter"
	}
	if m.focus == focusProjects && m.width < 100 {
		contextLabel = "PROJECTS  arrows  ·  Enter"
	}
	if m.focus == focusResults && m.width < 140 {
		contextLabel = "RESULTS  Up/Down rows  ·  Left/Right cols"
	}
	if m.focus == focusResults && m.width < 100 {
		contextLabel = "RESULTS  arrows rows/cols"
	}
	contextControls := contextStyle.Render(contextLabel)
	return lipgloss.JoinHorizontal(lipgloss.Top, tabControls, " ", contextControls)
}

func focusShortcutsLabel(current focus) string {
	switch current {
	case focusProjects:
		return "PROJECTS  Up/Down select  ·  Left/Right expand  ·  Enter info"
	case focusEditor:
		return "QUERY EDITOR  Ctrl+R run  ·  Enter newline"
	case focusResults:
		return "RESULTS  Up/Down rows  ·  Left/Right columns"
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
	panelHeight := max(10, m.height-15)
	boxStyle := m.panelBoxStyle(focusProjects).Padding(1).Width(m.projectPanelWidth()).Height(panelHeight)
	if len(m.projects) == 0 {
		content := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(muted).Render("No projects connected."),
			"",
			lipgloss.NewStyle().Foreground(accent).Render("No accessible projects found."),
		)
		return boxStyle.Render(content)
	}
	rows := m.projectRows()
	selectedRow := 0
	for index, row := range rows {
		if row.selected {
			selectedRow = index
			break
		}
	}
	viewportRows := max(1, panelHeight-4)
	maxStart := max(0, len(rows)-viewportRows)
	start := minInt(max(0, m.projectScroll), maxStart)
	if selectedRow < start {
		start = selectedRow
	} else if selectedRow >= start+viewportRows {
		start = selectedRow - viewportRows + 1
	}
	end := minInt(len(rows), start+viewportRows)
	lines := make([]string, 0, end-start)
	for _, row := range rows[start:end] {
		lines = append(lines, row.text)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return boxStyle.Render(content)
}

func (m model) selectedResourceView() string {
	if len(m.projects) == 0 || m.active < 0 || m.active >= len(m.projects) {
		return ""
	}
	name := m.projects[m.active].Name
	if name == "" {
		name = m.projects[m.active].ID
	}
	if m.selectedDataset >= 0 && m.selectedDataset < len(m.projects[m.active].Resources) {
		dataset := m.projects[m.active].Resources[m.selectedDataset]
		name = dataset.Name
		if m.selectedChild >= 0 && m.selectedChild < len(dataset.Children) {
			name = dataset.Children[m.selectedChild].Name
		}
	}
	lines := wrapText(name, max(8, m.projectPanelWidth()-2))
	for len(lines) < 2 {
		lines = append(lines, "")
	}
	return lipgloss.NewStyle().Foreground(ink).Width(m.projectPanelWidth()).Height(2).Render(strings.Join(lines, "\n"))
}

func (m model) projectRows() []projectRow {
	rows := []projectRow{}
	nameWidth := max(8, m.projectPanelWidth()-6)
	for i, item := range m.projects {
		marker := "  "
		selected := i == m.active && m.selectedDataset < 0
		if selected {
			marker = "▸ "
		}
		name := item.Name
		if name == "" {
			name = item.ID
		}
		line := marker + truncate(name, nameWidth)
		if selected {
			line = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(line)
		}
		rows = append(rows, projectRow{text: line, selected: selected})
		if i < len(m.expanded) && m.expanded[i] {
			if m.projectLoading[i] {
				rows = append(rows, projectRow{text: lipgloss.NewStyle().Foreground(muted).Render("    Loading datasets...")})
			}
			for resourceIndex, resource := range item.Resources {
				if resource.Kind != "dataset" {
					continue
				}
				marker := "    "
				selected := i == m.active && resourceIndex == m.selectedDataset && m.selectedChild < 0
				if selected {
					marker = "  ▸ "
				}
				line := marker + resourceIcon(resource.Kind) + " " + truncate(resource.Name, max(8, m.projectPanelWidth()-9))
				if selected {
					line = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(line)
				}
				rows = append(rows, projectRow{text: line, selected: selected})
				if i == m.active && m.datasetExpanded(i, resourceIndex) {
					if m.datasetLoading[m.datasetKey(i, resourceIndex)] {
						rows = append(rows, projectRow{text: lipgloss.NewStyle().Foreground(muted).Render("        Loading tables...")})
					}
					for childIndex, child := range resource.Children {
						childMarker := "        "
						selected := resourceIndex == m.selectedDataset && childIndex == m.selectedChild
						if selected {
							childMarker = "      ▸ "
						}
						childLine := childMarker + resourceIcon(child.Kind) + " " + truncate(child.Name, max(6, m.projectPanelWidth()-13))
						if selected {
							childLine = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(childLine)
						} else {
							childLine = lipgloss.NewStyle().Foreground(muted).Render(childLine)
						}
						rows = append(rows, projectRow{text: childLine, selected: selected})
					}
				}
			}
		}
	}
	return rows
}

func (m *model) updateProjectScroll() {
	if len(m.projects) == 0 {
		m.projectScroll = 0
		return
	}
	rows := m.projectRows()
	selectedRow := 0
	for index, row := range rows {
		if row.selected {
			selectedRow = index
			break
		}
	}
	viewportRows := max(1, max(10, m.height-15)-4)
	if selectedRow < m.projectScroll {
		m.projectScroll = selectedRow
	} else if selectedRow >= m.projectScroll+viewportRows {
		m.projectScroll = selectedRow - viewportRows + 1
	}
	maxStart := max(0, len(rows)-viewportRows)
	if m.projectScroll > maxStart {
		m.projectScroll = maxStart
	}
	if m.projectScroll < 0 {
		m.projectScroll = 0
	}
}

func (m model) panelBoxStyle(panelFocus focus) lipgloss.Style {
	color := border
	if m.focus == panelFocus {
		color = accent
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(color)
}

func (m model) infoView() string {
	lines := m.infoLines()
	modalWidth := max(40, m.width-4)
	modalHeight := max(12, m.height-4)
	viewport := max(1, modalHeight-6)
	maxScroll := max(0, len(lines)-viewport)
	scroll := m.infoScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	end := minInt(len(lines), scroll+viewport)
	visible := lines[scroll:end]
	visible = append(visible, "", lipgloss.NewStyle().Foreground(ink).Render("Up/Down scroll  ·  Enter/Esc close"))
	return lipgloss.NewStyle().Width(modalWidth).Height(modalHeight).Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(2).Render(lipgloss.JoinVertical(lipgloss.Left, visible...))
}

func (m model) infoLines() []string {
	title := "PROJECT"
	name := m.projects[m.active].Name
	identifier := m.projects[m.active].ID
	details := []string{"Project ID  " + identifier}
	if name == "" {
		name = identifier
	}
	if m.selectedDataset >= 0 {
		dataset := m.projects[m.active].Resources[m.selectedDataset]
		title = "DATASET"
		name = dataset.Name
		details = []string{"Project  " + m.projects[m.active].ID, fmt.Sprintf("Children  %d tables/views", len(dataset.Children))}
		if m.selectedChild >= 0 && m.selectedChild < len(dataset.Children) {
			child := dataset.Children[m.selectedChild]
			title = strings.ToUpper(child.Kind)
			name = child.Name
			details = []string{"Project  " + m.projects[m.active].ID, "Dataset  " + dataset.Name, "Type     " + child.Kind}
			if !child.DetailsLoaded {
				details = append(details, "Details  Loading...")
				return appendInfoLines([]string{panelTitle(title), "", lipgloss.NewStyle().Foreground(ink).Bold(true).Render(name), ""}, details)
			}
			details = append(details, resourceInfoLines(child)...)
			if len(child.ViewQuery) > 0 {
				details = append(details, "", "Query")
				for _, line := range wrapText(child.ViewQuery, max(20, m.width-10)) {
					details = append(details, "  "+line)
				}
			}
			if len(child.ExternalSource) > 0 {
				details = append(details, "", "External source")
				for _, source := range child.ExternalSource {
					details = append(details, "  "+truncate(source, 58))
				}
			}
			if len(child.Columns) > 0 {
				details = append(details, "", "Preview")
				details = append(details, formatPreview(child.Columns, child.Preview, 58)...)
			}
		}
	}
	return appendInfoLines([]string{panelTitle(title), "", lipgloss.NewStyle().Foreground(ink).Bold(true).Render(name), ""}, details)
}

func appendInfoLines(lines []string, details []string) []string {
	for _, detail := range details {
		lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render(detail))
	}
	return lines
}

func (m model) searchView() string {
	modalWidth := max(50, m.width-4)
	modalHeight := max(12, m.height-4)
	lines := []string{panelTitle("SEARCH RESOURCES"), "", m.searchInput.View(), ""}
	if len(m.searchResults) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render("No matching resources."))
	} else {
		maxRows := max(1, modalHeight-8)
		start := 0
		if m.searchCursor >= maxRows {
			start = m.searchCursor - maxRows + 1
		}
		end := minInt(len(m.searchResults), start+maxRows)
		for index := start; index < end; index++ {
			result := m.searchResults[index]
			location := result.project
			if result.dataset != "" {
				location += " / " + result.dataset
			}
			line := fmt.Sprintf("%-9s %-28s %s (%s)", strings.ToUpper(result.kind), truncate(result.name, 28), location, result.projectID)
			if index == m.searchCursor {
				line = lipgloss.NewStyle().Foreground(accent).Bold(true).Render("▸ " + line)
			}
			lines = append(lines, line)
		}
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(ink).Render("Up/Down select  ·  Enter open  ·  Esc close"))
	return lipgloss.NewStyle().Width(modalWidth).Height(modalHeight).Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func resourceInfoLines(resource project.Resource) []string {
	lines := []string{"", "Table info"}
	if resource.ID != "" {
		lines = append(lines, "  Table ID       "+resource.ID)
	}
	if !resource.Created.IsZero() {
		lines = append(lines, "  Created        "+resource.Created.Format(time.RFC3339))
	}
	if !resource.Modified.IsZero() {
		lines = append(lines, "  Last modified  "+resource.Modified.Format(time.RFC3339))
	}
	if resource.Expiration.IsZero() {
		lines = append(lines, "  Expiration     NEVER")
	} else {
		lines = append(lines, "  Expiration     "+resource.Expiration.Format(time.RFC3339))
	}
	if resource.Location != "" {
		lines = append(lines, "  Data location  "+resource.Location)
	}
	if resource.LegacySQL {
		lines = append(lines, "  Legacy SQL     true")
	} else {
		lines = append(lines, "  Legacy SQL     false")
	}
	if resource.Description != "" {
		lines = append(lines, "  Description    "+resource.Description)
	}
	if len(resource.Labels) > 0 {
		lines = append(lines, "  Labels         "+formatLabels(resource.Labels))
	}
	if resource.Kind == "table" || resource.Kind == "external" {
		lines = append(lines, "", "Storage info", fmt.Sprintf("  Number of rows          %d", resource.NumRows), "  Total logical bytes     "+formatBytes(resource.NumBytes), "  Long term logical bytes "+formatBytes(resource.LongTermBytes))
	}
	if resource.PartitionType != "" || resource.PartitionField != "" {
		lines = append(lines, "", "Partitioning")
		if resource.PartitionType != "" {
			lines = append(lines, "  Type             "+resource.PartitionType)
		}
		if resource.PartitionField != "" {
			lines = append(lines, "  Field            "+resource.PartitionField)
		}
		if resource.PartitionExpiration > 0 {
			lines = append(lines, "  Partition expiry "+resource.PartitionExpiration.String())
		}
		lines = append(lines, fmt.Sprintf("  Require filter   %t", resource.RequirePartitionFilter))
	}
	if len(resource.Clustering) > 0 {
		lines = append(lines, "", "Clustering", "  Fields           "+strings.Join(resource.Clustering, ", "))
	}
	return lines
}

func formatLabels(labels map[string]string) string {
	parts := make([]string, 0, len(labels))
	for key, value := range labels {
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ", ")
}

func formatPreview(columns []string, rows [][]string, width int) []string {
	if len(columns) == 0 {
		return nil
	}
	columnWidths := make([]int, len(columns))
	for index, column := range columns {
		columnWidths[index] = minInt(18, maxInt(4, len(column)))
	}
	for _, row := range rows {
		for index, value := range row {
			if index < len(columnWidths) {
				columnWidths[index] = minInt(18, maxInt(columnWidths[index], len(value)))
			}
		}
	}
	lines := []string{"  " + previewRow(columns, columnWidths), "  " + previewSeparator(columnWidths)}
	for _, row := range rows {
		lines = append(lines, "  "+previewRow(row, columnWidths))
	}
	return lines
}

func wrapText(value string, width int) []string {
	if width < 1 {
		return []string{value}
	}
	var lines []string
	for _, sourceLine := range strings.Split(value, "\n") {
		line := ""
		for _, character := range sourceLine {
			candidate := line + string(character)
			if line != "" && lipgloss.Width(candidate) > width {
				lines = append(lines, line)
				line = string(character)
			} else {
				line = candidate
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func previewRow(values []string, widths []int) string {
	parts := make([]string, len(widths))
	for index, width := range widths {
		value := ""
		if index < len(values) {
			value = truncate(values[index], width)
		}
		parts[index] = fmt.Sprintf("%-*s", width, value)
	}
	return strings.TrimRight(strings.Join(parts, "  "), " ")
}

func previewSeparator(widths []int) string {
	parts := make([]string, len(widths))
	for index, width := range widths {
		parts[index] = strings.Repeat("-", width)
	}
	return strings.Join(parts, "  ")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m model) editorView() string {
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("QUERY EDITOR")
	return lipgloss.JoinVertical(lipgloss.Left, title, m.panelBoxStyle(focusEditor).Background(panel).Render(m.tabs[m.activeTab].editor.View()))
}

func (m model) resultView() string {
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("RESULTS")
	resultHeight := max(3, m.tabs[m.activeTab].results.Height())
	resultWidth := max(10, m.tabs[m.activeTab].results.Width())
	return lipgloss.JoinVertical(lipgloss.Left, title, m.panelBoxStyle(focusResults).Width(resultWidth).Height(resultHeight).Render(m.renderResults()))
}

func (m model) renderResults() string {
	tab := m.tabs[m.activeTab]
	if len(tab.result.Columns) == 0 {
		return "Result"
	}
	width := tab.results.Width()
	rowNumberWidth := 5
	columnWidths := make([]int, len(tab.result.Columns))
	for index, column := range tab.result.Columns {
		columnWidths[index] = max(8, minInt(20, len(column)))
		for _, row := range tab.result.Rows {
			if index < len(row.Values) {
				columnWidths[index] = maxInt(columnWidths[index], minInt(20, len(row.Values[index])))
			}
		}
	}
	visibleWidth := 0
	offset := tab.resultOffset
	if tab.resultColumn >= offset {
		offset = tab.resultColumn
	}
	end := offset
	for end < len(columnWidths) && visibleWidth+columnWidths[end]+2 <= width-rowNumberWidth {
		visibleWidth += columnWidths[end] + 2
		end++
	}
	if end == offset {
		end++
	}
	lines := []string{"     " + resultRow(tab.result.Columns[offset:end], columnWidths[offset:end])}
	lines = append(lines, "     "+resultSeparator(columnWidths[offset:end]))
	viewport := max(1, tab.results.Height()-2)
	rowStart := 0
	if tab.resultRow >= viewport {
		rowStart = tab.resultRow - viewport + 1
	}
	rowEnd := minInt(len(tab.result.Rows), rowStart+viewport)
	for rowIndex := rowStart; rowIndex < rowEnd; rowIndex++ {
		row := tab.result.Rows[rowIndex]
		values := row.Values
		rowValues := resultWindow(values, offset, end)
		if tab.resultRow == rowIndex {
			lines = append(lines, lipgloss.NewStyle().Foreground(accent).Render(fmt.Sprintf("%4d ", rowIndex+1)+resultRow(rowValues, columnWidths[offset:end])))
		} else {
			lines = append(lines, fmt.Sprintf("%4d ", rowIndex+1)+resultRow(rowValues, columnWidths[offset:end]))
		}
	}
	return strings.Join(lines, "\n")
}

func resultRow(values []string, widths []int) string {
	parts := make([]string, len(widths))
	for index, width := range widths {
		value := ""
		if index < len(values) {
			value = truncate(values[index], width)
		}
		parts[index] = fmt.Sprintf("%-*s", width, value)
	}
	return strings.TrimRight(strings.Join(parts, "  "), " ")
}
func resultSeparator(widths []int) string {
	parts := make([]string, len(widths))
	for index, width := range widths {
		parts[index] = strings.Repeat("-", width)
	}
	return strings.Join(parts, "  ")
}
func resultWindow(values []string, start, end int) []string {
	window := make([]string, end-start)
	for index := range window {
		if start+index < len(values) {
			window[index] = values[start+index]
		}
	}
	return window
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
	return m.panelBoxStyle(focusHistory).Padding(1).Width(m.historyPanelWidth()).Height(max(10, m.height-15)).Render(content)
}

func (m model) projectPanelWidth() int { return max(18, minInt(24, m.width/5)) }
func (m model) historyPanelWidth() int { return max(22, minInt(30, m.width/4)) }

func panelTitle(title string) string {
	return lipgloss.NewStyle().Foreground(accent).Bold(true).Render(title)
}

func truncate(value string, width int) string {
	if width < 4 || len(value) <= width {
		return value
	}
	return value[:width-3] + "..."
}

func resourceIcon(kind string) string {
	switch kind {
	case "table":
		return "▣"
	case "external":
		return "⇄"
	case "view":
		return "◈"
	default:
		return "·"
	}
}

func formatBytes(value int64) string {
	if value < 1000 {
		return fmt.Sprintf("%d B", value)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	amount := float64(value)
	for _, unit := range units {
		amount /= 1000
		if amount < 1000 {
			return fmt.Sprintf("%.1f %s", amount, unit)
		}
	}
	return fmt.Sprintf("%.1f PB", amount/1000)
}

func formatValidationError(err error) string {
	message := err.Error()
	message = strings.TrimPrefix(message, "googleapi: Error 400: ")
	return strings.TrimSpace(message)
}

func (m model) helpView() string {
	lines := []string{"KEYMAP", "", "ctrl+s             search all resources", "tab / shift+tab   move focus", "ctrl+left/right   switch query tab", "ctrl+n             new query tab", "ctrl+w             close query tab", "up/down            select project or resource", "left/right         expand or collapse", "enter              inspect resource / newline", "ctrl+r             run query", "ctrl+enter         run when supported", "?                  close help", "q                  quit outside editor", "ctrl+c             quit"}
	return lipgloss.NewStyle().Width(50).Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(2).Render(strings.Join(lines, "\n"))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type tickMsg time.Time
