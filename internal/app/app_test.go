package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xjrr/bigtui/internal/bigquery"
	"github.com/xjrr/bigtui/internal/project"
)

type clientFunc func(context.Context, string, string) (bigquery.Result, error)

func (f clientFunc) Query(ctx context.Context, projectID, sql string) (bigquery.Result, error) {
	return f(ctx, projectID, sql)
}

func TestInitialQueryStartsEmpty(t *testing.T) {
	model := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	if value := model.tabs[0].editor.Value(); value != "" {
		t.Fatalf("expected an empty starter query, got %q", value)
	}
}

func TestMockInitializerLoadsFixtureCatalog(t *testing.T) {
	model := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	if len(model.projects) != 2 || len(model.projects[0].Resources) == 0 {
		t.Fatalf("expected mock projects and resources: %#v", model.projects)
	}
}

func TestProjectsTreeExpandsAndSelectsDatasets(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.focus = focusProjects
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyRight})
	state = updated.(model)
	if !state.expanded[0] {
		t.Fatal("right should expand the selected project")
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyDown})
	state = updated.(model)
	if state.selectedDataset < 0 || state.projects[state.active].Resources[state.selectedDataset].Kind != "dataset" {
		t.Fatalf("down should select a dataset: project=%d resource=%d", state.active, state.selectedDataset)
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyRight})
	state = updated.(model)
	if !state.datasetExpanded(state.active, state.selectedDataset) {
		t.Fatal("right should expand the selected dataset")
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyDown})
	state = updated.(model)
	if state.selectedChild < 0 {
		t.Fatal("down should select a table or view inside the dataset")
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyLeft})
	state = updated.(model)
	if state.selectedChild != -1 || state.selectedDataset == -1 {
		t.Fatal("left from a child should return to its dataset")
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyLeft})
	state = updated.(model)
	if state.datasetExpanded(state.active, state.selectedDataset) {
		t.Fatal("left from a dataset should collapse it")
	}
}

func TestVimHorizontalKeysExpandAndCollapseProjects(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.focus = focusProjects
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	state = updated.(model)
	if !state.expanded[0] {
		t.Fatal("l should expand the selected project")
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	state = updated.(model)
	if state.expanded[0] {
		t.Fatal("h should collapse the selected project")
	}
}

func TestEnterOpensResourceInfoModal(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.focus = focusProjects
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state = updated.(model)
	if !state.showInfo || !contains(state.infoView(), "Sandbox Analytics") {
		t.Fatal("enter should open project information")
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEscape})
	state = updated.(model)
	if state.showInfo {
		t.Fatal("escape should close resource information")
	}
	state.expanded[0] = true
	state.selectedDataset = 0
	state.expandedDataset[state.datasetKey(0, 0)] = true
	state.selectedChild = 1
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state = updated.(model)
	if !state.showInfo || !contains(state.infoView(), "customer_order_totals") {
		t.Fatal("enter should show selected child information")
	}
	state.showInfo = false
	state.selectedChild = 0
	state.showInfo = true
	if !contains(strings.Join(state.infoLines(), "\n"), "Ada Lovelace") || !contains(strings.Join(state.infoLines(), "\n"), "Preview") {
		t.Fatal("table information should include its data preview")
	}
	state.selectedChild = 2
	if !contains(strings.Join(state.infoLines(), "\n"), "gs://partner-feed/events/*.parquet") {
		t.Fatal("external table information should include its source")
	}
}

func TestPreviewIsStructuredAndModalFillsTerminal(t *testing.T) {
	preview := formatPreview([]string{"id", "name"}, [][]string{{"1", "Ada"}}, 30)
	if len(preview) != 3 || !contains(preview[0], "id") || !contains(preview[1], "-") || !contains(preview[2], "Ada") {
		t.Fatalf("unexpected structured preview: %#v", preview)
	}
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.width, state.height = 100, 30
	state.focus = focusProjects
	state.showInfo = true
	view := state.infoView()
	if lipgloss.Width(view) < 90 || lipgloss.Height(view) < 26 {
		t.Fatalf("modal should fill terminal: %dx%d", lipgloss.Width(view), lipgloss.Height(view))
	}
}

func TestViewQueryRemainsOneStyledBlock(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.selectedDataset = 0
	state.selectedChild = 1
	state.showInfo = true
	lines := state.infoLines()
	joined := strings.Join(lines, "\n")
	compact := strings.ReplaceAll(strings.ReplaceAll(joined, " ", ""), "\n", "")
	if !contains(compact, "SELECTcustomer_id") || !contains(compact, "GROUPBYcustomer_id") {
		t.Fatalf("view query block was not preserved: %q", joined)
	}
}

func TestWrapTextSplitsLongQueryLines(t *testing.T) {
	lines := wrapText("SELECT customer_id, customer_name FROM customer_order_totals", 20)
	if len(lines) < 2 {
		t.Fatalf("expected long query to wrap: %#v", lines)
	}
	for _, line := range lines {
		if lipgloss.Width(line) > 20 {
			t.Fatalf("wrapped query line exceeds width: %q", line)
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

func TestProjectIDIsTruncatedForSingleLineDisplay(t *testing.T) {
	value := truncate("project-33b62768-e4b9-483c-a1b", 18)
	if value != "project-33b6276..." {
		t.Fatalf("unexpected truncated project ID: %q", value)
	}
	if strings.Contains(value, "\n") {
		t.Fatalf("truncated project ID contains a newline: %q", value)
	}
}

func TestFormatBytes(t *testing.T) {
	for input, expected := range map[int64]string{0: "0 B", 1200: "1.2 KB", 1200000: "1.2 MB"} {
		if got := formatBytes(input); got != expected {
			t.Fatalf("formatBytes(%d) = %q, want %q", input, got, expected)
		}
	}
}

func TestFormatValidationErrorRemovesGoogleAPI400Prefix(t *testing.T) {
	message := formatValidationError(errors.New("googleapi: Error 400: Syntax error: Unexpected end of script at [1:1]"))
	if message != "Syntax error: Unexpected end of script at [1:1]" {
		t.Fatalf("unexpected validation error: %q", message)
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

func TestWorkspaceFitsWhenTerminalNarrows(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	for _, width := range []int{120, 100, 80} {
		updated, _ := state.Update(tea.WindowSizeMsg{Width: width, Height: 40})
		state = updated.(model)
		workspace := lipgloss.JoinHorizontal(lipgloss.Top, state.projectView(), "  ", state.editorView(), "  ", state.historyView())
		if got := lipgloss.Width(workspace); got > width || lipgloss.Width(state.shortcutView()) > width {
			t.Fatalf("workspace controls exceed terminal width %d: workspace=%d footer=%d", width, got, lipgloss.Width(state.shortcutView()))
		}
		state.setResult(0, bigquery.Result{Columns: []string{"customer_id", "name", "segment", "order_count", "lifetime_value"}, Rows: []bigquery.Row{{Values: []string{"1", "Ada Lovelace", "enterprise", "2", "4001/2"}}}})
		workspace = lipgloss.JoinHorizontal(lipgloss.Top, state.projectView(), "  ", state.editorView(), "  ", state.historyView())
		if got := lipgloss.Width(workspace); got > width {
			t.Fatalf("results expanded workspace beyond terminal width %d: rendered %d", width, got)
		}
	}
}

func TestResultsKeepHeadersAndRowNumbersWhileNavigating(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.focus = focusResults
	state.width, state.height = 100, 30
	state.resizeTab(0)
	state.setResult(0, bigquery.Result{Columns: []string{"id", "name", "segment"}, Rows: []bigquery.Row{{Values: []string{"1", "Ada", "enterprise"}}, {Values: []string{"2", "Grace", "startup"}}}})
	view := state.renderResults()
	if !contains(view, "id") || !contains(view, "name") || !contains(view, "1") {
		t.Fatalf("results should retain headers and row numbers: %q", view)
	}
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyDown})
	state = updated.(model)
	if state.tabs[0].resultRow != 1 {
		t.Fatalf("down should move result row cursor: %d", state.tabs[0].resultRow)
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	state = updated.(model)
	if state.tabs[0].resultColumn != 1 {
		t.Fatalf("l should move result column cursor: %d", state.tabs[0].resultColumn)
	}
}

func TestResultsViewportDoesNotGrowWithRows(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.width, state.height = 100, 30
	state.resizeTab(0)
	rows := make([]bigquery.Row, 100)
	for index := range rows {
		rows[index] = bigquery.Row{Values: []string{fmt.Sprint(index)}}
	}
	state.setResult(0, bigquery.Result{Columns: []string{"id"}, Rows: rows})
	if got, limit := lipgloss.Height(state.renderResults()), state.tabs[0].results.Height(); got > limit {
		t.Fatalf("results rendered %d rows beyond viewport %d", got, limit)
	}
}

func TestResultsPaneFillsAvailableHeight(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	updated, _ := state.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	state = updated.(model)
	if got, want := lipgloss.Height(state.resultView()), state.tabs[0].results.Height()+3; got != want {
		t.Fatalf("results pane height = %d, want %d", got, want)
	}
}

func TestCtrlJRecordsQueryRun(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	state.projects = project.MockProjects()
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
	state.projects = project.MockProjects()
	updated, command := state.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state = updated.(model)
	if command == nil || len(state.tabs[0].history) != 0 || state.tabs[0].editor.Value() != "\n" {
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
