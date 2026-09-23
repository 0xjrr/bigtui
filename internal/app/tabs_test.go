package app

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xjrr/bigtui/internal/naming"
)

func TestQueryTabsCanBeAddedSwitchedAndClosed(t *testing.T) {
	state := newModel()
	if len(state.tabs) != 1 || state.activeTab != 0 {
		t.Fatalf("unexpected initial tabs: %#v", state.tabs)
	}

	state.addTab()
	if len(state.tabs) != 2 || state.activeTab != 1 || !slices.Contains(naming.Cities, state.tabs[1].title) {
		t.Fatalf("tab was not added correctly: active=%d tabs=%#v", state.activeTab, state.tabs)
	}
	state.tabs[1].editor.SetValue("SELECT 2")
	state.switchTab(-1)
	if state.activeTab != 0 || state.tabs[state.activeTab].editor.Value() == "SELECT 2" {
		t.Fatal("switching tabs did not preserve separate query state")
	}
	state.switchTab(1)
	if state.tabs[state.activeTab].editor.Value() != "SELECT 2" {
		t.Fatal("returning to the new tab lost its query")
	}
	state.closeTab()
	if len(state.tabs) != 1 || state.activeTab != 0 {
		t.Fatalf("tab was not closed correctly: active=%d tabs=%#v", state.activeTab, state.tabs)
	}
}

func TestTabControlsWorkFromEditor(t *testing.T) {
	state, _ := press(newModel(), tea.KeyCtrlN)
	if len(state.tabs) != 2 {
		t.Fatalf("expected ctrl+n to add a tab from editor, got %d", len(state.tabs))
	}
	state, _ = press(state, tea.KeyCtrlLeft)
	if state.activeTab != 0 {
		t.Fatalf("expected ctrl+left to switch tabs from editor, got %d", state.activeTab)
	}
	state, _ = press(state, tea.KeyCtrlRight)
	if state.activeTab != 1 {
		t.Fatalf("expected ctrl+right to switch tabs from editor, got %d", state.activeTab)
	}
	state, _ = press(state, tea.KeyCtrlW)
	if len(state.tabs) != 1 || state.activeTab != 0 {
		t.Fatalf("expected ctrl+w to close the active tab: active=%d tabs=%d", state.activeTab, len(state.tabs))
	}
}

func TestSwitchingTabsWrapsAround(t *testing.T) {
	state := newModel()
	state.addTab()
	state.addTab()
	state.switchTab(1)
	if state.activeTab != 0 {
		t.Fatalf("switching past the last tab should wrap to the first: %d", state.activeTab)
	}
	state.switchTab(-1)
	if state.activeTab != 2 {
		t.Fatalf("switching before the first tab should wrap to the last: %d", state.activeTab)
	}
}

func TestTheLastTabCannotBeClosed(t *testing.T) {
	state := newModel()
	state.closeTab()
	if len(state.tabs) != 1 || state.status != "Cannot close the last tab." {
		t.Fatalf("unexpected close result: tabs=%d status=%q", len(state.tabs), state.status)
	}
}

func TestTabLabelsCompressEquallyWhenTheyExceedWidth(t *testing.T) {
	state := newModel()
	state.width = 40
	state.tabs = []queryTab{
		{title: "Auckland"},
		{title: "Buenos Aires"},
		{title: "Johannesburg"},
		{title: "San Francisco"},
	}
	view := state.tabView()
	if lipgloss.Width(view) > state.width {
		t.Fatalf("compressed tabs exceeded terminal width: %d > %d (%q)", lipgloss.Width(view), state.width, view)
	}
	if lipgloss.Width(view) != state.width {
		t.Fatalf("compressed tabs did not use the available width: %d != %d", lipgloss.Width(view), state.width)
	}
}

func TestNewTabUsesCurrentTerminalLayout(t *testing.T) {
	state, _ := send(newModel(), tea.WindowSizeMsg{Width: 120, Height: 40})
	firstHeight := lipgloss.Height(state.View())
	state.addTab()
	if got := lipgloss.Height(state.View()); got != firstHeight {
		t.Fatalf("new tab changed workspace height from %d to %d", firstHeight, got)
	}
}

func TestHistoryArrowDirectionMatchesReversedList(t *testing.T) {
	state := newModel()
	state.tabs[0].history = []runRecord{
		{sql: "SELECT 1", status: "done"},
		{sql: "SELECT 2", status: "done"},
		{sql: "SELECT 3", status: "done"},
	}
	state.tabs[0].historyCursor = 2
	state.focus = focusHistory

	state, _ = press(state, tea.KeyDown)
	if state.tabs[0].historyCursor != 1 {
		t.Fatalf("down should move to the next lower visible entry, got cursor %d", state.tabs[0].historyCursor)
	}
	state, _ = press(state, tea.KeyUp)
	if state.tabs[0].historyCursor != 2 {
		t.Fatalf("up should move to the previous higher visible entry, got cursor %d", state.tabs[0].historyCursor)
	}
}

func TestHistoryCursorStopsAtTheListBounds(t *testing.T) {
	state := newModel()
	state.tabs[0].history = []runRecord{{sql: "SELECT 1"}, {sql: "SELECT 2"}}
	state.tabs[0].historyCursor = 1
	state.focus = focusHistory
	for index := 0; index < 3; index++ {
		state, _ = press(state, tea.KeyDown)
	}
	if state.tabs[0].historyCursor != 0 {
		t.Fatalf("history cursor moved below the oldest run: %d", state.tabs[0].historyCursor)
	}
	for index := 0; index < 3; index++ {
		state, _ = press(state, tea.KeyUp)
	}
	if state.tabs[0].historyCursor != 1 {
		t.Fatalf("history cursor moved above the newest run: %d", state.tabs[0].historyCursor)
	}
}

func TestEnterLoadsTheSelectedHistoryQuery(t *testing.T) {
	state := newModel()
	state.tabs[0].history = []runRecord{{sql: "SELECT 1", status: "done"}, {sql: "SELECT 2", status: "done"}}
	state.tabs[0].historyCursor = 0
	state.focus = focusHistory
	state, _ = press(state, tea.KeyEnter)
	if state.focus != focusEditor {
		t.Fatalf("loading a run should focus the editor, got %s", focusLabel(state.focus))
	}
	if !strings.HasPrefix(state.tabs[0].editor.Value(), "SELECT 1") {
		t.Fatalf("editor should contain the selected run: %q", state.tabs[0].editor.Value())
	}
}

func TestHistoryViewListsRunsNewestFirst(t *testing.T) {
	state := newModel()
	state.width, state.height = 120, 40
	state.tabs[0].history = []runRecord{{sql: "SELECT 1", status: "done"}, {sql: "SELECT 2", status: "failed"}}
	view := state.historyView()
	if !strings.Contains(view, "SELECT 2") || !strings.Contains(view, "failed") {
		t.Fatalf("history view should list runs with their status: %q", view)
	}
	if strings.Index(view, "SELECT 2") > strings.Index(view, "SELECT 1") {
		t.Fatalf("history view should list the newest run first: %q", view)
	}
}

func TestHistoryViewReportsAnEmptyList(t *testing.T) {
	state := newModel()
	state.width, state.height = 120, 40
	if !strings.Contains(state.historyView(), "No runs yet") {
		t.Fatalf("empty history should be explained: %q", state.historyView())
	}
}
