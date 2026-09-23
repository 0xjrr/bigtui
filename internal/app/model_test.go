package app

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xjrr/bigtui/internal/bigquery"
	"github.com/xjrr/bigtui/internal/project"
)

func TestInitialQueryStartsEmpty(t *testing.T) {
	if value := newModel().tabs[0].editor.Value(); value != "" {
		t.Fatalf("expected an empty starter query, got %q", value)
	}
}

func TestInitialModelStartsInEditorFocusWithOneTab(t *testing.T) {
	state := newModel()
	if state.focus != focusEditor || len(state.tabs) != 1 || state.activeTab != 0 {
		t.Fatalf("unexpected initial state: focus=%v tabs=%d active=%d", state.focus, len(state.tabs), state.activeTab)
	}
	if state.selectedDataset != -1 || state.selectedChild != -1 {
		t.Fatalf("explorer selection should start empty: dataset=%d child=%d", state.selectedDataset, state.selectedChild)
	}
}

func TestMockInitializerLoadsFixtureCatalog(t *testing.T) {
	state := newMockModel()
	if len(state.projects) != 2 || len(state.projects[0].Resources) == 0 {
		t.Fatalf("expected mock projects and resources: %#v", state.projects)
	}
}

func TestInitWaitsForCatalogWhenALoaderIsConfigured(t *testing.T) {
	withLoader := newModelWithLoader(nil, &catalogLoaderStub{})
	if withLoader.Init() == nil || withLoader.status != "Loading projects..." {
		t.Fatalf("loader-backed model should announce catalog loading: %q", withLoader.status)
	}
	withProjects := newModelWithProjects(project.MockProjects())
	if withProjects.status != "Ready. Ctrl+R runs the query." {
		t.Fatalf("preloaded model should be ready: %q", withProjects.status)
	}
}

func TestFocusIndicatorCyclesThroughWorkspaceAreas(t *testing.T) {
	state := newModel()
	expected := []string{"QUERY EDITOR", "RESULTS", "RUN HISTORY", "SHORTCUTS", "EXPLORER"}
	for _, label := range expected {
		if got := focusLabel(state.focus); got != label {
			t.Fatalf("expected focus label %q, got %q", label, got)
		}
		updated, command := press(state, tea.KeyTab)
		state = updated
		if command != nil {
			t.Fatalf("tab should only change focus, got command %T", command)
		}
	}
}

func TestShiftTabCyclesFocusBackwards(t *testing.T) {
	state := newModel()
	expected := []focus{focusProjects, focusShortcuts, focusHistory, focusResults, focusEditor}
	for _, want := range expected {
		state, _ = send(state, tea.KeyMsg{Type: tea.KeyShiftTab})
		if state.focus != want {
			t.Fatalf("shift+tab moved to %s, want %s", focusLabel(state.focus), focusLabel(want))
		}
	}
}

func TestQQuitsFromWorkspace(t *testing.T) {
	state := newModel()
	state.focus = focusProjects
	state.applyFocus()
	_, command := typeRunes(state, 'q')
	if !isQuitCommand(command) {
		t.Fatal("expected q to return a quit command")
	}
}

func TestCtrlCQuitsFromAnyFocus(t *testing.T) {
	state := newModel()
	if _, command := press(state, tea.KeyCtrlC); !isQuitCommand(command) {
		t.Fatal("ctrl+c should quit from the editor")
	}
}

func TestEditorKeepsQAndQuestionMarkForSQL(t *testing.T) {
	state, command := typeRunes(newModel(), 'q')
	if isQuitCommand(command) || !strings.Contains(state.tabs[0].editor.Value(), "q") {
		t.Fatalf("q was intercepted by a global shortcut: quit=%v query=%q", isQuitCommand(command), state.tabs[0].editor.Value())
	}
	state, command = typeRunes(state, '?')
	if isQuitCommand(command) || !strings.Contains(state.tabs[0].editor.Value(), "?") || state.showHelp {
		t.Fatalf("question mark was intercepted by a global shortcut: quit=%v help=%v", isQuitCommand(command), state.showHelp)
	}
}

func TestCtrlBSelectsBillingProjectAndQueriesUseIt(t *testing.T) {
	queryProject := ""
	client := clientFunc(func(_ context.Context, projectID, _ string) (bigquery.Result, error) {
		queryProject = projectID
		return bigquery.Result{}, nil
	})
	state := initialModelWithProjects(client, []project.Project{{ID: "billing-project"}, {ID: "explorer-project"}})
	state.focus = focusProjects
	state.active = 1
	state, command := press(state, tea.KeyCtrlB)
	if command != nil || state.billingProject != 1 {
		t.Fatalf("Ctrl+B should select the active project for billing: billing=%d command=%v", state.billingProject, command != nil)
	}
	if state.billingProjectID() != "explorer-project" {
		t.Fatalf("unexpected billing project: %q", state.billingProjectID())
	}
	state.runQuery(-1)()
	if queryProject != "explorer-project" {
		t.Fatalf("query used project %q, want %q", queryProject, "explorer-project")
	}
}

func TestCtrlBIsIgnoredOutsideTheExplorer(t *testing.T) {
	state := newModelWithProjects([]project.Project{{ID: "first"}, {ID: "second"}})
	state.active = 1
	state, _ = press(state, tea.KeyCtrlB)
	if state.billingProject != 0 {
		t.Fatalf("editor focus should not change the billing project: %d", state.billingProject)
	}
}

func TestBillingProjectIDFallsBackToTheActiveProject(t *testing.T) {
	state := newModelWithProjects([]project.Project{{ID: "only-project"}})
	state.billingProject = -1
	state.active = 0
	if got := state.billingProjectID(); got != "only-project" {
		t.Fatalf("billing project fallback = %q, want %q", got, "only-project")
	}
	empty := newModelWithProjects(nil)
	if got := empty.billingProjectID(); got != "" {
		t.Fatalf("billing project without catalog = %q, want empty", got)
	}
}

func TestCtrlJRecordsQueryRun(t *testing.T) {
	state := newModel()
	state.projects = project.MockProjects()
	state, command := press(state, tea.KeyCtrlJ)
	if command == nil || len(state.tabs[0].history) != 1 {
		t.Fatalf("expected ctrl+j/ctrl+enter to start a run: command=%v history=%d", command != nil, len(state.tabs[0].history))
	}
	if state.tabs[0].history[0].status != "running" {
		t.Fatalf("unexpected run status: %q", state.tabs[0].history[0].status)
	}
}

func TestEnterAddsNewlineAndCtrlRRunsQuery(t *testing.T) {
	state := newModel()
	state.projects = project.MockProjects()
	state, command := press(state, tea.KeyEnter)
	if command == nil || len(state.tabs[0].history) != 0 || state.tabs[0].editor.Value() != "\n" {
		t.Fatalf("expected Enter to insert a newline: command=%v history=%d query=%q", command != nil, len(state.tabs[0].history), state.tabs[0].editor.Value())
	}
	state, command = press(state, tea.KeyCtrlR)
	if command == nil || len(state.tabs[0].history) != 1 {
		t.Fatalf("expected Ctrl+R to run a query: command=%v history=%d", command != nil, len(state.tabs[0].history))
	}
}

func TestRunningAQueryWithoutProjectsReportsStatus(t *testing.T) {
	state, command := press(newModelWithProjects(nil), tea.KeyCtrlR)
	if command != nil || state.status != "No project selected. Add a project first." {
		t.Fatalf("expected a guidance status without projects: command=%v status=%q", command != nil, state.status)
	}
}

func TestQueryResultsPopulateStatusAndHistory(t *testing.T) {
	state := newModelWithProjects(project.MockProjects())
	state.width, state.height = 100, 30
	state.resizeTab(0)
	state, _ = press(state, tea.KeyCtrlR)
	result := bigquery.Result{Columns: []string{"id"}, Rows: []bigquery.Row{{Values: []string{"1"}}}, Total: 1}
	state, _ = send(state, queryFinished{result: result, tab: 0, history: 0})
	if state.status != "Returned 1 rows" || state.tabs[0].history[0].status != "done" || state.tabs[0].history[0].rows != 1 {
		t.Fatalf("unexpected completion state: status=%q history=%#v", state.status, state.tabs[0].history)
	}
	failed := newModelWithProjects(project.MockProjects())
	failed, _ = press(failed, tea.KeyCtrlR)
	failed, _ = send(failed, queryFinished{err: errFailedQuery, tab: 0, history: 0})
	if !strings.HasPrefix(failed.status, "Query failed: ") || failed.tabs[0].history[0].status != "failed" {
		t.Fatalf("unexpected failure state: status=%q history=%#v", failed.status, failed.tabs[0].history)
	}
}

func TestQueryResultsForUnknownTabsAreIgnored(t *testing.T) {
	state := newModelWithProjects(project.MockProjects())
	state, _ = send(state, queryFinished{tab: 7, history: 0})
	if state.status != "Ready. Ctrl+R runs the query." {
		t.Fatalf("results for a closed tab should not change the status: %q", state.status)
	}
}

func TestAnalysisUpdatesTheValidationLine(t *testing.T) {
	state := newModelWithProjects(project.MockProjects())
	state, _ = send(state, queryAnalyzed{analysis: bigquery.Analysis{Valid: true, BytesProcessed: 1200}})
	if state.validation != "1 Valid · 1.2 KB processed" {
		t.Fatalf("unexpected validation line: %q", state.validation)
	}
	state, _ = send(state, queryAnalyzed{analysis: bigquery.Analysis{Err: errFailedQuery}})
	if !strings.HasPrefix(state.validation, "0 Invalid · ") {
		t.Fatalf("unexpected invalid validation line: %q", state.validation)
	}
}

func TestAnalyzeQueryOnlyRunsForAnalyzers(t *testing.T) {
	state := newModelWithProjects(project.MockProjects())
	if state.analyzeQuery() != nil {
		t.Fatal("clients without analysis support should not schedule a dry run")
	}
	analyzing := initialModelWithProjects(analyzerClient{clientFunc: stubClient(), analysis: bigquery.Analysis{Valid: true}}, project.MockProjects())
	command := analyzing.analyzeQuery()
	if command == nil {
		t.Fatal("analyzer clients should schedule a dry run")
	}
	if message, ok := command().(queryAnalyzed); !ok || !message.analysis.Valid {
		t.Fatalf("unexpected analysis message: %#v", message)
	}
}

var errFailedQuery = stubError("boom")

type stubError string

func (e stubError) Error() string { return string(e) }
