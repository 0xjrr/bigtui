package app

import (
	"context"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xjrr/bigtui/internal/completion"
	"github.com/0xjrr/bigtui/internal/ui/text"
	"github.com/0xjrr/bigtui/internal/ui/theme"
)

const (
	completionLimit    = 20
	completionRows     = 6
	completionMinWidth = 18
	completionMaxWidth = 36
	editorTopRow       = 6
)

func (m *model) closeCompletion() {
	m.completionOpen = false
	m.completionItems = nil
}

func (m *model) refreshCompletion() tea.Cmd {
	editor := m.tabs[m.activeTab].editor
	sql := editor.Value()
	cursor := cursorOffset(editor)
	parts := completion.ReferenceParts(sql, cursor)
	if completion.WordPrefix(sql, cursor) == "" && !qualifiedReference(parts) {
		m.closeCompletion()
		return nil
	}
	projectIndex := m.completionProjectIndex(parts)
	if command := m.ensureCatalogLoaded(projectIndex, parts, completion.IsResourceContext(sql, cursor)); command != nil {
		return command
	}
	m.setCompletionItems(m.completionItemsFor(completion.Request{
		Project: m.projectIDAt(projectIndex),
		SQL:     sql,
		Cursor:  cursor,
	}))
	return nil
}

func (m model) completionItemsFor(request completion.Request) []completion.Item {
	ctx := context.Background()
	catalogItems, _ := completion.CatalogProvider{Catalog: m.projects}.Complete(ctx, request)
	keywordItems, _ := completion.KeywordProvider{}.Complete(ctx, request)
	functionItems, _ := completion.FunctionProvider{}.Complete(ctx, request)
	items := completion.Merge(catalogItems, keywordItems, functionItems)
	return items[:min(len(items), completionLimit)]
}

func (m *model) setCompletionItems(items []completion.Item) {
	m.completionItems = items
	m.completionOpen = len(items) > 0
	if m.completionCursor >= len(items) {
		m.completionCursor = 0
	}
}

func (m model) completionProjectIndex(parts []string) int {
	if len(parts) < 2 {
		return m.active
	}
	for index, item := range m.projects {
		if item.ID == parts[0] {
			return index
		}
	}
	return m.active
}

func (m *model) ensureCatalogLoaded(projectIndex int, parts []string, resourceContext bool) tea.Cmd {
	if !resourceContext || m.loader == nil {
		return nil
	}
	if m.datasetsPending(projectIndex) {
		m.projectLoading[projectIndex] = true
		m.closeCompletion()
		m.status = "Loading datasets for autocomplete..."
		return m.loadDatasets(projectIndex)
	}
	datasetIndex, ok := m.completionDatasetIndex(projectIndex, parts)
	if !ok {
		return nil
	}
	key := datasetKey(projectIndex, datasetIndex)
	if m.tablesLoaded[key] || m.datasetLoading[key] {
		return nil
	}
	m.datasetLoading[key] = true
	m.closeCompletion()
	m.status = "Loading tables for autocomplete..."
	return m.loadTables(projectIndex, datasetIndex)
}

func (m model) datasetsPending(projectIndex int) bool {
	if projectIndex < 0 || projectIndex >= len(m.projects) {
		return false
	}
	return !m.datasetsLoaded[projectIndex] && !m.projectLoading[projectIndex]
}

func (m model) completionDatasetIndex(projectIndex int, parts []string) (int, bool) {
	if projectIndex < 0 || projectIndex >= len(m.projects) || len(parts) < 2 {
		return -1, false
	}
	datasetName := parts[0]
	if parts[0] == m.projects[projectIndex].ID {
		if len(parts) < 3 {
			return -1, false
		}
		datasetName = parts[1]
	}
	for index, resource := range m.projects[projectIndex].Resources {
		if resource.Kind == "dataset" && resource.Name == datasetName {
			return index, true
		}
	}
	return -1, false
}

func (m *model) acceptCompletion() tea.Cmd {
	if len(m.completionItems) == 0 {
		return nil
	}
	item := m.completionItems[m.completionCursor]
	editor := m.tabs[m.activeTab].editor
	value := editor.Value()
	cursor := cursorOffset(editor)
	prefix := completionPrefix(value, cursor)
	suffix := completionSuffix(item, completion.ReferenceIsQuoted(value, cursor))
	for range []rune(prefix) {
		editor, _ = editor.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	editor.InsertString(item.InsertText + suffix)
	m.tabs[m.activeTab].editor = editor
	m.closeCompletion()
	m.status = "Inserted " + item.Label
	if !strings.HasSuffix(suffix, ".") {
		return nil
	}
	return m.refreshCompletion()
}

func completionPrefix(value string, cursor int) string {
	if prefix := completion.WordPrefix(value, cursor); prefix != "" {
		return prefix
	}
	parts := completion.ReferenceParts(value, cursor)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func completionSuffix(item completion.Item, quoted bool) string {
	if item.Detail == "project" || strings.HasPrefix(item.Detail, "dataset ·") {
		return "."
	}
	if quoted && hasAnyPrefix(item.Detail, "table ·", "view ·", "external ·") {
		return "`"
	}
	return ""
}

func qualifiedReference(parts []string) bool {
	return len(parts) > 1 && parts[len(parts)-1] == ""
}

func hasAnyPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func cursorOffset(editor textarea.Model) int {
	lines := strings.Split(editor.Value(), "\n")
	row := clamp(editor.Line(), 0, len(lines)-1)
	offset := 0
	for index := 0; index < row; index++ {
		offset += len([]rune(lines[index])) + 1
	}
	info := editor.LineInfo()
	return offset + min(info.StartColumn+info.ColumnOffset, len([]rune(lines[row])))
}

func (m model) cursorScreenPosition() (int, int) {
	editor := m.tabs[m.activeTab].editor
	info := editor.LineInfo()
	gutterWidth := 0
	if editor.ShowLineNumbers {
		gutterWidth = len(strconv.Itoa(editor.MaxHeight)) + 2
	}
	column := lipgloss.Width(editor.Prompt) + gutterWidth + info.StartColumn + info.ColumnOffset
	row := editorTopRow + editor.Line() + info.RowOffset
	return row, lipgloss.Width(m.projectView()) + 3 + column
}

func (m model) overlayCompletion(view string) string {
	popup := m.completionView()
	popupWidth := lipgloss.Width(popup)
	popupHeight := lipgloss.Height(popup)
	row, column := m.cursorScreenPosition()
	row++
	if column+popupWidth > m.width {
		column = max(0, m.width-popupWidth)
	}
	lines := strings.Split(view, "\n")
	if row+popupHeight > len(lines) {
		row = max(0, len(lines)-popupHeight)
	}
	return text.Overlay(view, popup, row, column)
}

func (m model) completionView() string {
	start := 0
	if m.completionCursor >= completionRows {
		start = m.completionCursor - completionRows + 1
	}
	end := min(len(m.completionItems), start+completionRows)
	width := m.completionWidth(start, end)
	lines := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		item := m.completionItems[index]
		line := text.Truncate(item.Label+"  "+item.Detail, width-2)
		if index == m.completionCursor {
			lines = append(lines, theme.Selected("▸ "+line))
			continue
		}
		lines = append(lines, theme.Dim("  "+line))
	}
	return lipgloss.NewStyle().Width(width).Border(lipgloss.RoundedBorder()).BorderForeground(theme.Accent).Background(theme.Panel).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) completionWidth(start, end int) int {
	width := completionMinWidth
	for index := start; index < end; index++ {
		item := m.completionItems[index]
		width = max(width, lipgloss.Width(item.Label)+lipgloss.Width(item.Detail)+5)
	}
	return min(width, completionMaxWidth)
}
