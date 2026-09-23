package app

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/0xjrr/bigtui/internal/bigquery"
	"github.com/0xjrr/bigtui/internal/completion"
	"github.com/0xjrr/bigtui/internal/naming"
	"github.com/0xjrr/bigtui/internal/project"
)

type focus int

const (
	focusProjects focus = iota
	focusEditor
	focusResults
	focusHistory
	focusShortcuts

	focusCount = 5
)

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
	client             bigquery.Client
	loader             project.CatalogLoader
	projects           []project.Project
	active             int
	billingProject     int
	focus              focus
	tabs               []queryTab
	activeTab          int
	status             string
	validation         string
	width              int
	height             int
	showHelp           bool
	helpScroll         int
	expanded           []bool
	selectedDataset    int
	selectedChild      int
	expandedDataset    map[string]bool
	showInfo           bool
	infoScroll         int
	searchOpen         bool
	searchInput        textinput.Model
	searchResults      []searchResult
	searchCursor       int
	resourceLoading    bool
	projectLoading     map[int]bool
	datasetLoading     map[string]bool
	datasetsLoaded     map[int]bool
	tablesLoaded       map[string]bool
	projectScroll      int
	showHiddenDatasets bool
	completionOpen     bool
	completionItems    []completion.Item
	completionCursor   int
}

func New(client bigquery.Client) *tea.Program {
	return NewWithProjects(client, nil)
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

func NewWithLoader(client bigquery.Client, loader project.CatalogLoader) *tea.Program {
	return NewWithProjectsAndLoader(client, nil, loader)
}

func NewWithProjectsAndLoader(client bigquery.Client, projects []project.Project, loader project.CatalogLoader) *tea.Program {
	return tea.NewProgram(initialModelWithProjectsAndLoader(client, projects, loader), tea.WithAltScreen())
}

func initialModel(client bigquery.Client) model {
	return initialModelWithMock(client, mockDataRequested())
}

func initialModelWithMock(client bigquery.Client, mock bool) model {
	projects := []project.Project{}
	if mock || mockDataRequested() {
		projects = project.MockProjects()
	}
	return initialModelWithProjects(client, projects)
}

func initialModelWithProjects(client bigquery.Client, projects []project.Project) model {
	return initialModelWithProjectsAndLoader(client, projects, nil)
}

func initialModelWithProjectsAndLoader(client bigquery.Client, projects []project.Project, loader project.CatalogLoader) model {
	tab := newQueryTab(naming.RandomCity(), "")
	tab.editor.Focus()
	return model{
		client:          client,
		loader:          loader,
		projects:        projects,
		focus:           focusEditor,
		tabs:            []queryTab{tab},
		status:          initialStatus(loader, projects),
		expanded:        make([]bool, len(projects)),
		selectedDataset: -1,
		selectedChild:   -1,
		expandedDataset: map[string]bool{},
		projectLoading:  map[int]bool{},
		datasetLoading:  map[string]bool{},
		datasetsLoaded:  map[int]bool{},
		tablesLoaded:    map[string]bool{},
		searchInput:     newSearchInput(),
	}
}

func (m model) Init() tea.Cmd {
	if m.catalogPending() {
		return tea.Batch(textarea.Blink, m.loadProjects())
	}
	return textarea.Blink
}

func (m model) catalogPending() bool {
	return m.loader != nil && len(m.projects) == 0
}

func (m *model) activeQueryTab() *queryTab {
	return &m.tabs[m.activeTab]
}

func (m *model) applyFocus() {
	m.closeCompletion()
	for index := range m.tabs {
		m.tabs[index].editor.Blur()
		m.tabs[index].results.Blur()
	}
	if m.focus == focusEditor {
		m.tabs[m.activeTab].editor.Focus()
	}
	m.tabs[m.activeTab].results.SetCursor(0)
}

func (m model) billingProjectID() string {
	if m.billingProject >= 0 && m.billingProject < len(m.projects) {
		return m.projects[m.billingProject].ID
	}
	return m.projectIDAt(m.active)
}

func (m model) projectIDAt(index int) string {
	if index < 0 || index >= len(m.projects) {
		return ""
	}
	return m.projects[index].ID
}

func (m model) pageStep() int {
	return max(1, m.height/2)
}

func mockDataRequested() bool {
	return os.Getenv("BIGTUI_MOCK_DATA") == "1"
}

func initialStatus(loader project.CatalogLoader, projects []project.Project) string {
	if loader != nil && len(projects) == 0 {
		return "Loading projects..."
	}
	return "Ready. Ctrl+R runs the query."
}

func newSearchInput() textinput.Model {
	input := textinput.New()
	input.Placeholder = "Search projects, datasets, tables, and views..."
	input.CharLimit = 120
	return input
}
