package app

import (
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/xjrr/bigtui/internal/naming"
)

func newQueryTab(title, sql string) queryTab {
	editor := textarea.New()
	editor.Placeholder = "Write SQL..."
	editor.SetValue(sql)
	editor.Prompt = "  "
	editor.CharLimit = 10000
	editor.ShowLineNumbers = true
	editor.SetHeight(7)
	results := table.New(table.WithColumns([]table.Column{{Title: "Result", Width: 22}}), table.WithFocused(false))
	return queryTab{title: title, editor: editor, results: results}
}

func (m *model) addTab() {
	title := naming.RandomCity()
	m.tabs = append(m.tabs, newQueryTab(title, ""))
	m.activeTab = len(m.tabs) - 1
	m.resizeTab(m.activeTab)
	m.applyFocus()
	m.status = "Opened " + title
}

func (m *model) closeTab() {
	if len(m.tabs) == 1 {
		m.status = "Cannot close the last tab."
		return
	}
	closed := m.tabs[m.activeTab].title
	m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
	m.activeTab = min(m.activeTab, len(m.tabs)-1)
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

func (m *model) resizeTab(index int) {
	if index < 0 || index >= len(m.tabs) {
		return
	}
	contentWidth := max(10, m.width-m.projectPanelWidth()-m.historyPanelWidth()-m.layoutReserve())
	m.tabs[index].editor.SetWidth(contentWidth)
	m.tabs[index].results.SetWidth(contentWidth)
	m.resizeResultColumns(index)
	m.tabs[index].results.SetHeight(max(3, m.height-23))
}

func (m model) layoutReserve() int {
	switch {
	case m.width < 90:
		return 29
	case m.width < 110:
		return 19
	default:
		return 14
	}
}

func (m *model) recordQuery() int {
	tab := m.activeQueryTab()
	tab.history = append(tab.history, runRecord{sql: tab.editor.Value(), started: time.Now(), status: "running"})
	tab.historyCursor = len(tab.history) - 1
	return tab.historyCursor
}

func (m *model) moveHistoryCursor(direction int) {
	tab := m.activeQueryTab()
	if direction > 0 && tab.historyCursor > 0 {
		tab.historyCursor--
		return
	}
	if direction < 0 && tab.historyCursor < len(tab.history)-1 {
		tab.historyCursor++
	}
}

func (m *model) loadHistoryQuery() {
	tab := m.activeQueryTab()
	if len(tab.history) == 0 {
		return
	}
	tab.editor.SetValue(tab.history[tab.historyCursor].sql)
	m.focus = focusEditor
	m.applyFocus()
	m.status = "Loaded query from run history"
}
