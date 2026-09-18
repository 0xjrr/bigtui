package app

import (
	"context"

	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xjrr/bigtui/internal/bigquery"
)

type clientFunc func(context.Context, string, string) (bigquery.Result, error)

func (f clientFunc) Query(ctx context.Context, projectID, sql string) (bigquery.Result, error) {
	return f(ctx, projectID, sql)
}

func TestInitialQueryUsesRealLineBreaks(t *testing.T) {
	model := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	if value := model.tabs[0].editor.Value(); value == "" || value[0] == '\\' || value[0] == ' ' {
		t.Fatalf("unexpected starter query: %q", value)
	}
	for _, line := range []string{"FROM `demo-analytics", "GROUP BY project_id", "ORDER BY rows DESC"} {
		if !contains(model.tabs[0].editor.Value(), line) {
			t.Fatalf("starter query is missing %q: %q", line, model.tabs[0].editor.Value())
		}
	}
}

func TestQQuitsFromWorkspace(t *testing.T) {
	model := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	model.focus = focusProjects
	model.applyFocus()
	_, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if command == nil {
		t.Fatal("expected q to return a quit command")
	}
	if _, ok := command().(tea.QuitMsg); !ok {
		t.Fatalf("expected quit message, got %T", command())
	}
}

func TestEditorKeepsQAndQuestionMarkForSQL(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	updated, command := state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	state = updated.(model)
	if isQuitCommand(command) || !contains(state.tabs[0].editor.Value(), "q") {
		t.Fatalf("q was intercepted by a global shortcut: quit=%v query=%q", isQuitCommand(command), state.tabs[0].editor.Value())
	}
	updated, command = state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	state = updated.(model)
	if isQuitCommand(command) || !contains(state.tabs[0].editor.Value(), "?") || state.showHelp {
		t.Fatalf("question mark was intercepted by a global shortcut: quit=%v help=%v", isQuitCommand(command), state.showHelp)
	}
}

func TestTabControlsWorkFromEditor(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	state = updated.(model)
	if len(state.tabs) != 2 {
		t.Fatalf("expected ctrl+n to add a tab from editor, got %d", len(state.tabs))
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyCtrlLeft})
	state = updated.(model)
	if state.activeTab != 0 {
		t.Fatalf("expected ctrl+left to switch tabs from editor, got %d", state.activeTab)
	}
}

func TestFocusIndicatorCyclesThroughWorkspaceAreas(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	expected := []string{"QUERY EDITOR", "RESULTS", "RUN HISTORY", "SHORTCUTS", "PROJECTS"}
	for _, label := range expected {
		if got := focusLabel(state.focus); got != label {
			t.Fatalf("expected focus label %q, got %q", label, got)
		}
		var command tea.Cmd
		updated, command := state.Update(tea.KeyMsg{Type: tea.KeyTab})
		state = updated.(model)
		if command != nil {
			t.Fatalf("tab should only change focus, got command %T", command)
		}
	}
}

func TestShortcutFocusDoesNotResizeWorkspace(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	updated, _ := state.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	state = updated.(model)
	initialHeight := lipgloss.Height(state.View())
	if initialHeight > state.height {
		t.Fatalf("workspace exceeds terminal height: %d > %d", initialHeight, state.height)
	}
	for index := 0; index < 3; index++ {
		updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyTab})
		state = updated.(model)
	}
	if state.focus != focusShortcuts {
		t.Fatalf("expected shortcut focus, got %s", focusLabel(state.focus))
	}
	if got := lipgloss.Height(state.View()); got != initialHeight {
		t.Fatalf("shortcut focus changed workspace height from %d to %d", initialHeight, got)
	}
}

func TestCtrlJRecordsQueryRun(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	updated, command := state.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	state = updated.(model)
	if command == nil || len(state.tabs[0].history) != 1 {
		t.Fatalf("expected ctrl+j/ctrl+enter to start a run: command=%v history=%d", command != nil, len(state.tabs[0].history))
	}
	if state.tabs[0].history[0].status != "running" {
		t.Fatalf("unexpected run status: %q", state.tabs[0].history[0].status)
	}
}

func TestEnterAddsNewlineAndCtrlRRunsQuery(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	updated, command := state.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state = updated.(model)
	if command == nil || len(state.tabs[0].history) != 0 || !contains(state.tabs[0].editor.Value(), "ORDER BY rows DESC\n") {
		t.Fatalf("expected Enter to insert a newline: command=%v history=%d query=%q", command != nil, len(state.tabs[0].history), state.tabs[0].editor.Value())
	}

	updated, command = state.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	state = updated.(model)
	if command == nil || len(state.tabs[0].history) != 1 {
		t.Fatalf("expected Ctrl+R to run a query: command=%v history=%d", command != nil, len(state.tabs[0].history))
	}
}

func TestQueryTabsCanBeAddedSwitchedAndClosed(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	if len(state.tabs) != 1 || state.activeTab != 0 {
		t.Fatalf("unexpected initial tabs: %#v", state.tabs)
	}

	state.addTab()
	if len(state.tabs) != 2 || state.activeTab != 1 || state.tabs[1].title != "Query 2" {
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

func TestNewTabUsesCurrentTerminalLayout(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	updated, _ := state.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	state = updated.(model)
	firstHeight := lipgloss.Height(state.View())
	state.addTab()
	if got := lipgloss.Height(state.View()); got != firstHeight {
		t.Fatalf("new tab changed workspace height from %d to %d", firstHeight, got)
	}
}

func TestHistoryArrowDirectionMatchesReversedList(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	state.tabs[0].history = []runRecord{
		{sql: "SELECT 1", status: "done"},
		{sql: "SELECT 2", status: "done"},
		{sql: "SELECT 3", status: "done"},
	}
	state.tabs[0].historyCursor = 2
	state.focus = focusHistory

	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyDown})
	state = updated.(model)
	if state.tabs[0].historyCursor != 1 {
		t.Fatalf("down should move to the next lower visible entry, got cursor %d", state.tabs[0].historyCursor)
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyUp})
	state = updated.(model)
	if state.tabs[0].historyCursor != 2 {
		t.Fatalf("up should move to the previous higher visible entry, got cursor %d", state.tabs[0].historyCursor)
	}
}

func contains(value, part string) bool {
	for index := 0; index+len(part) <= len(value); index++ {
		if value[index:index+len(part)] == part {
			return true
		}
	}
	return false
}

func isQuitCommand(command tea.Cmd) bool {
	if command == nil {
		return false
	}
	_, ok := command().(tea.QuitMsg)
	return ok
}
