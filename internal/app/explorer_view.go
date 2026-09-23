package app

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/xjrr/bigtui/internal/project"
	"github.com/xjrr/bigtui/internal/ui/text"
	"github.com/xjrr/bigtui/internal/ui/theme"
)

func (m model) projectView() string {
	box := theme.Box(m.focus == focusProjects).Padding(1).Width(m.projectPanelWidth()).Height(m.explorerPanelHeight())
	if len(m.projects) == 0 {
		return box.Render(lipgloss.JoinVertical(lipgloss.Left,
			theme.Dim("No projects connected."),
			"",
			theme.Emphasis("No accessible projects found."),
		))
	}
	rows := m.projectRows()
	start := m.explorerScrollStart(rows)
	end := min(len(rows), start+m.explorerViewportRows())
	lines := make([]string, 0, end-start)
	for _, row := range rows[start:end] {
		lines = append(lines, row.text)
	}
	return box.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) explorerPanelHeight() int {
	return max(10, m.height-15)
}

func (m model) explorerViewportRows() int {
	return max(1, m.explorerPanelHeight()-4)
}

func (m model) explorerScrollStart(rows []projectRow) int {
	selected := selectedRowIndex(rows)
	viewport := m.explorerViewportRows()
	start := min(max(0, m.projectScroll), max(0, len(rows)-viewport))
	if selected < start {
		return selected
	}
	if selected >= start+viewport {
		return selected - viewport + 1
	}
	return start
}

func (m *model) updateProjectScroll() {
	if len(m.projects) == 0 {
		m.projectScroll = 0
		return
	}
	rows := m.projectRows()
	selected := selectedRowIndex(rows)
	viewport := m.explorerViewportRows()
	if selected < m.projectScroll {
		m.projectScroll = selected
	} else if selected >= m.projectScroll+viewport {
		m.projectScroll = selected - viewport + 1
	}
	m.projectScroll = min(m.projectScroll, max(0, len(rows)-viewport))
	m.projectScroll = max(0, m.projectScroll)
}

func (m model) projectRows() []projectRow {
	rows := []projectRow{}
	for index, item := range m.projects {
		rows = append(rows, m.projectHeaderRow(index, item))
		rows = append(rows, m.datasetRows(index, item)...)
	}
	return rows
}

func (m model) projectHeaderRow(index int, item project.Project) projectRow {
	selected := index == m.active && m.selectedDataset < 0
	name := item.Name
	if name == "" {
		name = item.ID
	}
	return newProjectRow(marker(selected, "  ", "▸ ")+text.Truncate(name, max(8, m.projectPanelWidth()-6)), selected)
}

func (m model) datasetRows(projectIndex int, item project.Project) []projectRow {
	if projectIndex >= len(m.expanded) || !m.expanded[projectIndex] {
		return nil
	}
	rows := []projectRow{}
	if m.projectLoading[projectIndex] {
		rows = append(rows, projectRow{text: theme.Dim("    Loading datasets...")})
	}
	for resourceIndex, resource := range item.Resources {
		if resource.Kind != "dataset" {
			continue
		}
		selected := projectIndex == m.active && resourceIndex == m.selectedDataset && m.selectedChild < 0
		line := marker(selected, "    ", "  ▸ ") + resourceIcon(resource.Kind) + " " + text.Truncate(resource.Name, max(8, m.projectPanelWidth()-9))
		rows = append(rows, newProjectRow(line, selected))
		rows = append(rows, m.childRows(projectIndex, resourceIndex, resource)...)
	}
	return rows
}

func (m model) childRows(projectIndex, datasetIndex int, dataset project.Resource) []projectRow {
	if projectIndex != m.active || !m.datasetExpanded(projectIndex, datasetIndex) {
		return nil
	}
	rows := []projectRow{}
	if m.datasetLoading[datasetKey(projectIndex, datasetIndex)] {
		rows = append(rows, projectRow{text: theme.Dim("        Loading tables...")})
	}
	for childIndex, child := range dataset.Children {
		selected := datasetIndex == m.selectedDataset && childIndex == m.selectedChild
		line := marker(selected, "        ", "      ▸ ") + resourceIcon(child.Kind) + " " + text.Truncate(child.Name, max(6, m.projectPanelWidth()-13))
		if !selected {
			line = theme.Dim(line)
		}
		rows = append(rows, newProjectRow(line, selected))
	}
	return rows
}

func newProjectRow(line string, selected bool) projectRow {
	if selected {
		return projectRow{text: theme.Selected(line), selected: true}
	}
	return projectRow{text: line}
}

func selectedRowIndex(rows []projectRow) int {
	for index, row := range rows {
		if row.selected {
			return index
		}
	}
	return 0
}

func marker(selected bool, plain, highlighted string) string {
	if selected {
		return highlighted
	}
	return plain
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
