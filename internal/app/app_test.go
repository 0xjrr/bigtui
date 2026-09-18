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
	_, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if command == nil {
		t.Fatal("expected q to return a quit command")
	}
	if _, ok := command().(tea.QuitMsg); !ok {
		t.Fatalf("expected quit message, got %T", command())
	}
}

func TestFocusIndicatorCyclesThroughWorkspaceAreas(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	expected := []string{"QUERY EDITOR", "RESULTS", "SHORTCUTS", "PROJECTS"}
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
	for index := 0; index < 2; index++ {
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

func contains(value, part string) bool {
	for index := 0; index+len(part) <= len(value); index++ {
		if value[index:index+len(part)] == part {
			return true
		}
	}
	return false
}
