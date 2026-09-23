package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xjrr/bigtui/internal/project"
	"github.com/0xjrr/bigtui/internal/ui/text"
	"github.com/0xjrr/bigtui/internal/ui/theme"
)

func (m *model) openSearch() {
	m.searchOpen = true
	m.searchInput.SetValue("")
	m.searchInput.Focus()
	m.refreshSearch()
}

func (m *model) handleSearchKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "esc", "ctrl+s":
		m.searchOpen = false
		m.searchInput.Blur()
		return nil, true
	case "enter":
		m.selectSearchResult()
		return nil, true
	case "up", "ctrl+k":
		m.searchCursor = max(0, m.searchCursor-1)
		return nil, true
	case "down", "ctrl+j":
		if m.searchCursor < len(m.searchResults)-1 {
			m.searchCursor++
		}
		return nil, true
	}
	var command tea.Cmd
	m.searchInput, command = m.searchInput.Update(msg)
	m.refreshSearch()
	return command, true
}

func (m *model) refreshSearch() {
	term := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	m.searchResults = nil
	for projectIndex, item := range m.projects {
		m.searchResults = append(m.searchResults, searchProject(term, projectIndex, item)...)
	}
	if m.searchCursor >= len(m.searchResults) {
		m.searchCursor = max(0, len(m.searchResults)-1)
	}
}

func searchProject(term string, projectIndex int, item project.Project) []searchResult {
	name := item.Name
	if name == "" {
		name = item.ID
	}
	results := []searchResult{}
	if matchesSearch(term, name, item.ID) {
		results = append(results, searchResult{projectIndex: projectIndex, datasetIndex: -1, childIndex: -1, kind: "project", name: name, project: name, projectID: item.ID})
	}
	for datasetIndex, resource := range item.Resources {
		if resource.Kind != "dataset" {
			continue
		}
		if matchesSearch(term, resource.Name, name, item.ID) {
			results = append(results, searchResult{projectIndex: projectIndex, datasetIndex: datasetIndex, childIndex: -1, kind: "dataset", name: resource.Name, project: name, projectID: item.ID})
		}
		for childIndex, child := range resource.Children {
			if !matchesSearch(term, child.Name, name, item.ID, resource.Name) {
				continue
			}
			results = append(results, searchResult{projectIndex: projectIndex, datasetIndex: datasetIndex, childIndex: childIndex, kind: child.Kind, name: child.Name, project: name, projectID: item.ID, dataset: resource.Name})
		}
	}
	return results
}

func matchesSearch(term, name string, context ...string) bool {
	if term == "" {
		return true
	}
	for _, value := range append([]string{name}, context...) {
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
		m.expandedDataset[datasetKey(result.projectIndex, result.datasetIndex)] = result.childIndex >= 0
	}
	m.searchOpen = false
	m.searchInput.Blur()
	m.focus = focusProjects
	m.applyFocus()
	m.status = "Selected " + result.name
}

func (m model) searchView() string {
	width := max(50, m.width-4)
	height := max(12, m.height-4)
	lines := []string{theme.Title("SEARCH RESOURCES"), "", m.searchInput.View(), ""}
	lines = append(lines, m.searchResultLines(height)...)
	lines = append(lines, "", theme.Bright("Up/Down select  ·  Enter open  ·  Esc close"))
	return theme.Modal(width, height).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) searchResultLines(modalHeight int) []string {
	if len(m.searchResults) == 0 {
		return []string{theme.Dim("No matching resources.")}
	}
	maxRows := max(1, modalHeight-8)
	start := 0
	if m.searchCursor >= maxRows {
		start = m.searchCursor - maxRows + 1
	}
	end := min(len(m.searchResults), start+maxRows)
	lines := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		line := searchResultLine(m.searchResults[index])
		if index == m.searchCursor {
			line = theme.Selected("▸ " + line)
		}
		lines = append(lines, line)
	}
	return lines
}

func searchResultLine(result searchResult) string {
	location := result.project
	if result.dataset != "" {
		location += " / " + result.dataset
	}
	return fmt.Sprintf("%-9s %-28s %s (%s)", strings.ToUpper(result.kind), text.Truncate(result.name, 28), location, result.projectID)
}
