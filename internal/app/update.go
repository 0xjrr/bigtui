package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/0xjrr/bigtui/internal/format"
	"github.com/0xjrr/bigtui/internal/ui/text"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(message.Width, message.Height)
	case queryFinished:
		if m.applyQueryResult(message) {
			return m, nil
		}
	case queryAnalyzed:
		m.applyAnalysis(message)
	case projectsLoaded:
		if m.applyProjects(message) {
			return m, nil
		}
	case datasetsLoaded:
		if command, stop := m.applyDatasets(message); stop {
			return m, command
		}
	case tablesLoaded:
		if command, stop := m.applyTables(message); stop {
			return m, command
		}
	case resourceLoaded:
		if m.applyResource(message) {
			return m, nil
		}
	case tea.KeyMsg:
		if command, stop := m.handleKey(message); stop {
			return m, command
		}
	}
	return m, m.forwardToFocus(msg)
}

func (m *model) forwardToFocus(msg tea.Msg) tea.Cmd {
	tab := m.activeQueryTab()
	var command tea.Cmd
	switch m.focus {
	case focusEditor:
		tab.editor, command = tab.editor.Update(msg)
		if _, ok := msg.(tea.KeyMsg); !ok {
			return command
		}
		return tea.Batch(command, m.refreshCompletion(), m.analyzeQuery())
	case focusResults:
		tab.results, command = tab.results.Update(msg)
	}
	return command
}

func (m *model) resize(width, height int) {
	m.width, m.height = width, height
	for index := range m.tabs {
		m.resizeTab(index)
	}
	m.updateProjectScroll()
}

func (m *model) applyQueryResult(msg queryFinished) bool {
	if msg.tab < 0 || msg.tab >= len(m.tabs) {
		return true
	}
	m.recordQueryOutcome(msg)
	if msg.err != nil {
		m.status = "Query failed: " + msg.err.Error()
		return false
	}
	m.status = fmt.Sprintf("Returned %d rows", msg.result.Total)
	m.setResult(msg.tab, msg.result)
	return false
}

func (m *model) recordQueryOutcome(msg queryFinished) {
	if msg.history < 0 || msg.history >= len(m.tabs[msg.tab].history) {
		return
	}
	record := &m.tabs[msg.tab].history[msg.history]
	if msg.err != nil {
		record.status = "failed"
		return
	}
	record.status = "done"
	record.rows = msg.result.Total
}

func (m *model) applyAnalysis(msg queryAnalyzed) {
	if msg.analysis.Err != nil {
		m.validation = "0 Invalid · " + text.Truncate(format.ValidationError(msg.analysis.Err), 70)
		return
	}
	if !msg.analysis.Valid {
		return
	}
	m.validation = fmt.Sprintf("1 Valid · %s processed", format.Bytes(msg.analysis.BytesProcessed))
}

func (m *model) applyProjects(msg projectsLoaded) bool {
	if msg.err != nil {
		m.status = "Project loading failed: " + msg.err.Error()
		return true
	}
	m.projects = msg.projects
	m.expanded = make([]bool, len(msg.projects))
	m.billingProject = 0
	m.selectedDataset = -1
	m.selectedChild = -1
	m.projectLoading = map[int]bool{}
	m.datasetLoading = map[string]bool{}
	m.datasetsLoaded = map[int]bool{}
	m.tablesLoaded = map[string]bool{}
	m.projectScroll = 0
	m.updateProjectScroll()
	m.status = fmt.Sprintf("Ready. Loaded %d projects.", len(msg.projects))
	return false
}

func (m *model) applyDatasets(msg datasetsLoaded) (tea.Cmd, bool) {
	if msg.includeHidden != m.showHiddenDatasets {
		return nil, true
	}
	m.projectLoading[msg.projectIndex] = false
	if msg.err != nil {
		m.status = "Dataset loading failed: " + msg.err.Error()
		return nil, true
	}
	if msg.projectIndex < 0 || msg.projectIndex >= len(m.projects) {
		return nil, false
	}
	m.projects[msg.projectIndex].Resources = msg.resources
	m.datasetsLoaded[msg.projectIndex] = true
	m.updateProjectScroll()
	m.status = fmt.Sprintf("Loaded %d datasets.", len(msg.resources))
	if m.focus != focusEditor {
		return nil, false
	}
	return m.refreshCompletion(), true
}

func (m *model) applyTables(msg tablesLoaded) (tea.Cmd, bool) {
	key := datasetKey(msg.projectIndex, msg.datasetIndex)
	m.datasetLoading[key] = false
	if msg.err != nil {
		m.status = "Table loading failed: " + msg.err.Error()
		return nil, true
	}
	if !m.datasetExists(msg.projectIndex, msg.datasetIndex) {
		return nil, false
	}
	dataset := &m.projects[msg.projectIndex].Resources[msg.datasetIndex]
	dataset.Children = msg.resources
	dataset.ChildrenLoaded = true
	m.tablesLoaded[key] = true
	m.updateProjectScroll()
	m.status = fmt.Sprintf("Loaded %d tables/views.", len(msg.resources))
	if m.focus != focusEditor {
		return nil, false
	}
	return m.refreshCompletion(), true
}

func (m *model) applyResource(msg resourceLoaded) bool {
	m.resourceLoading = false
	if msg.err != nil {
		m.status = "Resource details failed: " + msg.err.Error()
		return true
	}
	if !m.datasetExists(msg.projectIndex, msg.datasetIndex) {
		return false
	}
	dataset := &m.projects[msg.projectIndex].Resources[msg.datasetIndex]
	if msg.childIndex < 0 || msg.childIndex >= len(dataset.Children) {
		return false
	}
	dataset.Children[msg.childIndex] = msg.resource
	m.status = "Loaded details for " + msg.resource.Name
	return false
}

func (m model) datasetExists(projectIndex, datasetIndex int) bool {
	if projectIndex < 0 || projectIndex >= len(m.projects) {
		return false
	}
	return datasetIndex >= 0 && datasetIndex < len(m.projects[projectIndex].Resources)
}
