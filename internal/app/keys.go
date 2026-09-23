package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) handleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	key := msg.String()
	switch {
	case m.searchOpen:
		return m.handleSearchKey(msg)
	case m.showInfo:
		m.handleInfoKey(key)
		return nil, true
	case m.showHelp:
		m.handleHelpKey(key)
		return nil, true
	case key == "ctrl+c", key == "q" && m.focus != focusEditor:
		return tea.Quit, true
	case key == "?" && m.focus != focusEditor:
		m.showHelp = !m.showHelp
		m.helpScroll = 0
		return nil, true
	}
	if command, stop := m.handleCompletionKey(key); stop {
		return command, true
	}
	if command, stop := m.handleWorkspaceKey(key); stop {
		return command, true
	}
	return m.handleFocusKey(key)
}

func (m *model) handleInfoKey(key string) {
	switch key {
	case "esc", "enter", "q":
		m.showInfo = false
		m.infoScroll = 0
	case "up", "k":
		m.infoScroll = max(0, m.infoScroll-1)
	case "down", "j":
		m.infoScroll++
	case "pgup":
		m.infoScroll = max(0, m.infoScroll-m.pageStep())
	case "pgdown":
		m.infoScroll += m.pageStep()
	}
}

func (m *model) handleHelpKey(key string) {
	switch key {
	case "esc", "q", "?":
		m.showHelp = false
		m.helpScroll = 0
	case "up", "k":
		m.helpScroll = max(0, m.helpScroll-1)
	case "down", "j":
		m.helpScroll++
	case "pgup":
		m.helpScroll = max(0, m.helpScroll-m.pageStep())
	case "pgdown":
		m.helpScroll += m.pageStep()
	}
}

func (m *model) handleCompletionKey(key string) (tea.Cmd, bool) {
	if !m.completionOpen || m.focus != focusEditor {
		return nil, false
	}
	switch key {
	case "esc":
		m.closeCompletion()
		return nil, true
	case "up", "ctrl+k":
		m.completionCursor = max(0, m.completionCursor-1)
		return nil, true
	case "down", "ctrl+j":
		if m.completionCursor < len(m.completionItems)-1 {
			m.completionCursor++
		}
		return nil, true
	case "enter", "tab":
		return m.acceptCompletion(), true
	case "left", "right":
		m.closeCompletion()
	}
	return nil, false
}

func (m *model) handleWorkspaceKey(key string) (tea.Cmd, bool) {
	switch key {
	case "ctrl+s":
		m.openSearch()
		return nil, true
	case "ctrl+h":
		if m.focus != focusProjects {
			return nil, false
		}
		return m.toggleHiddenDatasets(), true
	case "ctrl+b":
		if m.focus != focusProjects || m.active < 0 || m.active >= len(m.projects) {
			return nil, false
		}
		m.billingProject = m.active
		m.status = "Billing project: " + m.projects[m.billingProject].ID
		return nil, true
	case "ctrl+e":
		if m.focus != focusProjects {
			return nil, false
		}
		m.insertSelectedReference()
		return nil, true
	case "ctrl+n":
		m.addTab()
		return nil, true
	case "ctrl+w":
		m.closeTab()
		return nil, true
	case "alt+left", "ctrl+left", "ctrl+shift+tab":
		m.switchTab(-1)
		return nil, true
	case "alt+right", "ctrl+right", "ctrl+tab":
		m.switchTab(1)
		return nil, true
	}
	return nil, false
}

func (m *model) handleFocusKey(key string) (tea.Cmd, bool) {
	switch key {
	case "tab":
		m.moveFocus(1)
		return nil, true
	case "shift+tab":
		m.moveFocus(-1)
		return nil, true
	case "ctrl+r", "ctrl+enter", "ctrl+j":
		return m.runFocusedQuery()
	case "enter":
		return m.confirmSelection()
	case "j", "down":
		m.moveSelection(1)
	case "k", "up":
		m.moveSelection(-1)
	case "left", "h":
		m.moveOut()
	case "right", "l":
		if m.focus == focusProjects {
			return m.expandProject(), true
		}
		if m.focus == focusResults {
			m.moveResultColumn(1)
		}
	}
	return nil, false
}

func (m *model) moveFocus(direction int) {
	m.focus = (m.focus + focus(direction) + focusCount) % focusCount
	m.applyFocus()
}

func (m *model) moveSelection(direction int) {
	switch m.focus {
	case focusProjects:
		m.moveProjectSelection(direction)
	case focusHistory:
		m.moveHistoryCursor(direction)
	case focusResults:
		m.moveResultRow(direction)
	}
}

func (m *model) moveOut() {
	switch m.focus {
	case focusProjects:
		m.collapseProject()
	case focusResults:
		m.moveResultColumn(-1)
	}
}

func (m *model) runFocusedQuery() (tea.Cmd, bool) {
	if m.focus != focusEditor {
		return nil, false
	}
	if len(m.projects) == 0 {
		m.status = "No project selected. Add a project first."
		return nil, true
	}
	m.status = "Running query against " + m.billingProjectID() + "..."
	return m.runQuery(m.recordQuery()), true
}

func (m *model) confirmSelection() (tea.Cmd, bool) {
	switch m.focus {
	case focusProjects:
		if len(m.projects) == 0 {
			return nil, false
		}
		m.showInfo = true
		m.infoScroll = 0
		return m.loadSelectedResource(), true
	case focusHistory:
		m.loadHistoryQuery()
	}
	return nil, false
}

func (m *model) loadSelectedResource() tea.Cmd {
	if m.selectedDataset < 0 || m.selectedChild < 0 {
		return nil
	}
	child := m.projects[m.active].Resources[m.selectedDataset].Children[m.selectedChild]
	if child.DetailsLoaded || m.loader == nil || m.resourceLoading {
		return nil
	}
	m.resourceLoading = true
	m.status = "Loading details for " + child.Name + "..."
	return m.loadResource(m.active, m.selectedDataset, m.selectedChild)
}
