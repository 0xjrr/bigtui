package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/xjrr/bigtui/internal/ui/text"
	"github.com/xjrr/bigtui/internal/ui/theme"
)

func (m model) View() string {
	switch {
	case m.width == 0:
		return "Starting bigtui..."
	case m.showInfo:
		return m.infoView()
	case m.searchOpen:
		return m.searchView()
	case m.showHelp:
		return m.helpView()
	}
	explorer := lipgloss.JoinVertical(lipgloss.Left, theme.Title("EXPLORER"), m.projectView())
	main := lipgloss.JoinVertical(lipgloss.Left, m.editorView(), m.resultView())
	history := lipgloss.JoinVertical(lipgloss.Left, theme.Title("RUN HISTORY"), m.historyView())
	workspace := lipgloss.JoinHorizontal(lipgloss.Top, explorer, "  ", main, "  ", history)
	view := lipgloss.JoinVertical(lipgloss.Left,
		m.headerView(),
		m.tabView(),
		theme.Title("FOCUS: "+focusLabel(m.focus)),
		"",
		workspace,
		m.contextInfoView(),
		theme.Emphasis("● "+m.status),
		theme.Dim(m.validation),
		m.shortcutView(),
	)
	if m.completionOpen && len(m.completionItems) > 0 {
		return m.overlayCompletion(view)
	}
	return view
}

func (m model) headerView() string {
	title := theme.Strong("BIGTUI") + "  " + theme.Dim("BigQuery workspace")
	billing := "BILLING: none"
	if id := m.billingProjectID(); id != "" {
		billing = "BILLING: " + id
	}
	billingView := theme.Dim(billing)
	spacing := max(1, m.width-lipgloss.Width(title)-lipgloss.Width(billingView))
	return title + strings.Repeat(" ", spacing) + billingView
}

func (m model) tabView() string {
	labels := make([]string, len(m.tabs))
	naturalWidth := 0
	for index, tab := range m.tabs {
		labels[index] = fmt.Sprintf("%d %s", index+1, tab.title)
		naturalWidth += lipgloss.Width(labels[index]) + 2
	}
	compressed := naturalWidth > m.width && m.width > 0
	items := make([]string, 0, len(labels))
	for index, label := range labels {
		items = append(items, m.tabLabel(index, label, len(labels), compressed))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, items...)
}

func (m model) tabLabel(index int, label string, count int, compressed bool) string {
	style := lipgloss.NewStyle().Foreground(theme.Muted).Padding(0, 1)
	if index == m.activeTab {
		style = style.Foreground(theme.Ink).Bold(true).Background(theme.Panel).Underline(true)
	}
	if !compressed {
		return style.Render(label)
	}
	width := max(1, m.width/count)
	if index < m.width%count {
		width++
	}
	return style.Width(width).MaxHeight(1).Render(ansi.Truncate(label, max(1, width-2), ""))
}

func (m model) editorView() string {
	box := theme.Box(m.focus == focusEditor).Background(theme.Panel).Render(m.tabs[m.activeTab].editor.View())
	return lipgloss.JoinVertical(lipgloss.Left, theme.Title("QUERY EDITOR"), box)
}

func (m model) historyView() string {
	tab := m.tabs[m.activeTab]
	lines := []string{}
	for index := len(tab.history) - 1; index >= 0; index-- {
		lines = append(lines, historyLine(tab, index))
	}
	if len(lines) == 0 {
		lines = append(lines, theme.Dim("  No runs yet"))
	}
	return theme.Box(m.focus == focusHistory).Padding(1).Width(m.historyPanelWidth()).Height(max(10, m.height-15)).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func historyLine(tab queryTab, index int) string {
	record := tab.history[index]
	query := strings.Join(strings.Fields(record.sql), " ")
	if len(query) > 25 {
		query = query[:25] + "..."
	}
	selected := index == tab.historyCursor
	line := fmt.Sprintf("%s%d %s [%s]", marker(selected, "  ", "▸ "), index+1, query, record.status)
	if selected {
		return theme.Selected(line)
	}
	return line
}

func (m model) contextInfoView() string {
	name, detail := m.contextInfo()
	width := max(8, m.width-2)
	style := lipgloss.NewStyle().Foreground(theme.Ink).Width(width).Height(2)
	if name == "" {
		return style.Render("")
	}
	lines := text.Wrap(name, width)
	if detail != "" {
		lines = append(lines, text.Wrap(detail, width)...)
	}
	lines = lines[:min(len(lines), 2)]
	for len(lines) < 2 {
		lines = append(lines, "")
	}
	return style.Render(strings.Join(lines, "\n"))
}

func (m model) contextInfo() (string, string) {
	if m.focus == focusEditor && m.completionOpen && len(m.completionItems) > 0 {
		item := m.completionItems[m.completionCursor]
		return item.Label, item.Detail
	}
	if len(m.projects) == 0 || m.active < 0 || m.active >= len(m.projects) {
		return "", ""
	}
	name := m.projects[m.active].ID
	if !m.datasetExists(m.active, m.selectedDataset) {
		return name, ""
	}
	dataset := m.projects[m.active].Resources[m.selectedDataset]
	name += "." + dataset.Name
	if m.selectedChild >= 0 && m.selectedChild < len(dataset.Children) {
		name += "." + dataset.Children[m.selectedChild].Name
	}
	return name, ""
}

func (m model) shortcutView() string {
	tabStyle := lipgloss.NewStyle().Foreground(theme.Ink).Border(lipgloss.RoundedBorder()).BorderForeground(theme.Border).Padding(0, 1)
	contextStyle := lipgloss.NewStyle().Foreground(theme.Muted).Border(lipgloss.RoundedBorder()).BorderForeground(theme.Panel).Padding(0, 1)
	if m.focus == focusShortcuts {
		contextStyle = contextStyle.Foreground(theme.Ink).Bold(true).BorderForeground(theme.Accent)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		tabStyle.Render(tabShortcutLabel(m.width)),
		" ",
		contextStyle.Render(contextShortcutLabel(m.focus, m.width)),
	)
}

func tabShortcutLabel(width int) string {
	switch {
	case width < 100:
		return "TABS  Ctrl+Left/Right"
	case width < 140:
		return "TABS  Ctrl+Left/Right  ·  Ctrl+N  ·  Ctrl+W"
	default:
		return "TABS  Ctrl+Left/Right switch  ·  Ctrl+N new  ·  Ctrl+W close  ·  Tab focus"
	}
}

func contextShortcutLabel(current focus, width int) string {
	switch {
	case current == focusProjects && width < 100:
		return "EXPLORER  arrows  ·  Ctrl+B billing  ·  Ctrl+E insert"
	case current == focusProjects && width < 140:
		return "EXPLORER  arrows  ·  Ctrl+B billing  ·  Ctrl+E insert  ·  Ctrl+H hidden  ·  Enter"
	case current == focusResults && width < 100:
		return "RESULTS  arrows rows/cols"
	case current == focusResults && width < 140:
		return "RESULTS  Up/Down rows  ·  Left/Right cols"
	case current == focusEditor && width < 100:
		return "EDITOR  Ctrl+R"
	case current == focusEditor && width < 140:
		return "EDITOR  Ctrl+R run  ·  auto-complete"
	default:
		return focusShortcutsLabel(current)
	}
}

func focusShortcutsLabel(current focus) string {
	switch current {
	case focusProjects:
		return "EXPLORER  Up/Down select  ·  Ctrl+B set billing project  ·  Left/Right expand  ·  Ctrl+E query  ·  Ctrl+H hidden  ·  Enter info"
	case focusEditor:
		return "QUERY EDITOR  Ctrl+R run  ·  auto-complete as you type  ·  Enter newline"
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
		return "EXPLORER"
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

func (m model) projectPanelWidth() int { return max(18, min(30, m.width/4)) }

func (m model) historyPanelWidth() int { return max(22, min(30, m.width/4)) }
