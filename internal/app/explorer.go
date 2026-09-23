package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func datasetKey(projectIndex, resourceIndex int) string {
	return fmt.Sprintf("%d:%d", projectIndex, resourceIndex)
}

func (m model) datasetExpanded(projectIndex, resourceIndex int) bool {
	return m.expandedDataset[datasetKey(projectIndex, resourceIndex)]
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

func (m *model) moveProjectSelection(direction int) {
	defer m.updateProjectScroll()
	if len(m.projects) == 0 {
		return
	}
	if m.enterExpandedProject(direction) {
		return
	}
	if m.selectedDataset >= 0 && m.moveWithinProject(direction) {
		return
	}
	m.moveAcrossProjects(direction)
}

func (m *model) enterExpandedProject(direction int) bool {
	if direction < 0 || m.selectedDataset >= 0 || m.active >= len(m.expanded) || !m.expanded[m.active] {
		return false
	}
	datasets := m.datasetIndexes(m.active)
	if len(datasets) == 0 {
		return false
	}
	m.selectedDataset = datasets[0]
	return true
}

func (m *model) moveWithinProject(direction int) bool {
	if m.selectedChild >= 0 && m.moveWithinChildren(direction) {
		return true
	}
	if m.enterExpandedDataset(direction) {
		return true
	}
	datasets := m.datasetIndexes(m.active)
	position := indexOf(datasets, m.selectedDataset)
	if direction > 0 && position < len(datasets)-1 {
		m.selectDataset(datasets[position+1])
		return true
	}
	if direction < 0 && position > 0 {
		m.selectDataset(datasets[position-1])
		return true
	}
	if direction < 0 {
		m.selectDataset(-1)
		return true
	}
	return false
}

func (m *model) moveWithinChildren(direction int) bool {
	children := m.projects[m.active].Resources[m.selectedDataset].Children
	if direction > 0 && m.selectedChild < len(children)-1 {
		m.selectedChild++
		return true
	}
	if direction < 0 && m.selectedChild > 0 {
		m.selectedChild--
		return true
	}
	if direction < 0 {
		m.selectedChild = -1
		return true
	}
	return false
}

func (m *model) enterExpandedDataset(direction int) bool {
	if direction < 0 || m.selectedChild >= 0 || !m.datasetExpanded(m.active, m.selectedDataset) {
		return false
	}
	if len(m.projects[m.active].Resources[m.selectedDataset].Children) == 0 {
		return false
	}
	m.selectedChild = 0
	return true
}

func (m *model) moveAcrossProjects(direction int) {
	switch {
	case direction > 0 && m.active < len(m.projects)-1:
		m.active++
	case direction < 0 && m.active > 0:
		m.active--
	default:
		return
	}
	m.selectDataset(-1)
}

func (m *model) selectDataset(index int) {
	m.selectedDataset = index
	m.selectedChild = -1
}

func (m *model) expandProject() tea.Cmd {
	defer m.updateProjectScroll()
	if m.selectedDataset >= 0 {
		return m.expandDataset()
	}
	if m.active < 0 || m.active >= len(m.expanded) {
		return nil
	}
	m.expanded[m.active] = true
	if m.loader == nil || m.datasetsLoaded[m.active] || m.projectLoading[m.active] {
		return nil
	}
	m.projectLoading[m.active] = true
	m.status = "Loading datasets..."
	return m.loadDatasets(m.active)
}

func (m *model) expandDataset() tea.Cmd {
	key := datasetKey(m.active, m.selectedDataset)
	m.expandedDataset[key] = true
	if m.loader == nil || m.tablesLoaded[key] || m.datasetLoading[key] {
		return nil
	}
	m.datasetLoading[key] = true
	m.status = "Loading tables and views..."
	return m.loadTables(m.active, m.selectedDataset)
}

func (m *model) collapseProject() {
	defer m.updateProjectScroll()
	if m.selectedChild >= 0 {
		m.selectedChild = -1
		return
	}
	if m.selectedDataset >= 0 {
		m.collapseDataset()
		return
	}
	if m.active >= 0 && m.active < len(m.expanded) {
		m.expanded[m.active] = false
	}
}

func (m *model) collapseDataset() {
	key := datasetKey(m.active, m.selectedDataset)
	if m.expandedDataset[key] {
		delete(m.expandedDataset, key)
		return
	}
	m.selectedDataset = -1
}

func (m *model) toggleHiddenDatasets() tea.Cmd {
	if m.loader == nil {
		m.status = "Hidden dataset filtering is unavailable in mock mode."
		return nil
	}
	m.showHiddenDatasets = !m.showHiddenDatasets
	m.datasetsLoaded = map[int]bool{}
	m.projectLoading = map[int]bool{}
	m.datasetLoading = map[string]bool{}
	m.tablesLoaded = map[string]bool{}
	m.expandedDataset = map[string]bool{}
	m.selectDataset(-1)
	commands := m.reloadExpandedProjects()
	m.updateProjectScroll()
	m.status = "Hiding hidden datasets..."
	if m.showHiddenDatasets {
		m.status = "Loading hidden datasets..."
	}
	if len(commands) == 0 {
		return nil
	}
	return tea.Batch(commands...)
}

func (m *model) reloadExpandedProjects() []tea.Cmd {
	commands := []tea.Cmd{}
	for index := range m.projects {
		m.projects[index].Resources = nil
		if index >= len(m.expanded) || !m.expanded[index] {
			continue
		}
		m.projectLoading[index] = true
		commands = append(commands, m.loadDatasets(index))
	}
	return commands
}

func (m *model) insertSelectedReference() {
	if !m.datasetExists(m.active, m.selectedDataset) {
		return
	}
	dataset := m.projects[m.active].Resources[m.selectedDataset]
	if dataset.Kind != "dataset" {
		return
	}
	name := m.projects[m.active].ID + "." + dataset.Name
	if m.selectedChild >= 0 && m.selectedChild < len(dataset.Children) {
		name += "." + dataset.Children[m.selectedChild].Name
	}
	m.activeQueryTab().editor.InsertString("SELECT * FROM `" + name + "`")
	m.focus = focusEditor
	m.applyFocus()
	m.status = "Inserted " + name
}

func indexOf(values []int, target int) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return 0
}
