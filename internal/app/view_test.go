package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xjrr/bigtui/internal/bigquery"
	"github.com/xjrr/bigtui/internal/completion"
	"github.com/xjrr/bigtui/internal/project"
)

func TestViewWaitsForTheFirstWindowSize(t *testing.T) {
	if got := newModel().View(); got != "Starting bigtui..." {
		t.Fatalf("unexpected startup view: %q", got)
	}
}

func TestHeaderShowsBillingProjectAtRight(t *testing.T) {
	state := newModelWithProjects([]project.Project{{ID: "billing-project"}})
	state.width = 80
	if !strings.Contains(state.headerView(), "BILLING: billing-project") {
		t.Fatalf("header should show billing project: %q", state.headerView())
	}
	if lipgloss.Width(state.headerView()) != state.width {
		t.Fatalf("header width = %d, want %d", lipgloss.Width(state.headerView()), state.width)
	}
}

func TestHeaderReportsMissingBillingProject(t *testing.T) {
	state := newModelWithProjects(nil)
	state.width = 80
	if !strings.Contains(state.headerView(), "BILLING: none") {
		t.Fatalf("header should report a missing billing project: %q", state.headerView())
	}
}

func TestProjectContextShowsFullNameWithoutBorder(t *testing.T) {
	state := newModelWithProjects([]project.Project{{
		ID:        "project-id",
		Name:      "project-name",
		Resources: []project.Resource{{Name: "dataset-name", Kind: "dataset", Children: []project.Resource{{Name: "a-very-long-table-name", Kind: "table"}}}},
	}})
	state.width = 80
	state.active = 0
	state.selectedDataset = 0
	state.selectedChild = 0
	view := state.contextInfoView()
	compact := strings.ReplaceAll(strings.ReplaceAll(view, " ", ""), "\n", "")
	if !strings.Contains(compact, "project-id.dataset-name.a-very-long-table-name") {
		t.Fatalf("selected resource path is missing: %q", view)
	}
	if strings.Contains(view, "...") || strings.Contains(view, "─") || strings.Contains(view, "│") {
		t.Fatalf("selected resource view should be untruncated and unbordered: %q", view)
	}
}

func TestDatasetContextShowsFullyQualifiedName(t *testing.T) {
	state := newModelWithProjects([]project.Project{{ID: "project-id", Resources: []project.Resource{{Name: "dataset-name", Kind: "dataset"}}}})
	state.width = 80
	state.focus = focusProjects
	state.selectedDataset = 0
	state.selectedChild = -1
	if got := strings.ReplaceAll(strings.ReplaceAll(state.contextInfoView(), " ", ""), "\n", ""); !strings.Contains(got, "project-id.dataset-name") {
		t.Fatalf("dataset context should be fully qualified: %q", got)
	}
}

func TestEditorContextShowsSelectedCompletionDescription(t *testing.T) {
	state := newModel()
	state.width = 80
	state.focus = focusEditor
	state.completionOpen = true
	state.completionItems = []completion.Item{{Label: "S2_CELLIDFROMPOINT", Detail: "Gets the S2 cell ID covering a point GEOGRAPHY value."}}
	view := state.contextInfoView()
	if !strings.Contains(view, "S2_CELLIDFROMPOINT") || !strings.Contains(view, "Gets the S2 cell ID") {
		t.Fatalf("editor context should show selected completion details: %q", view)
	}
	if strings.Contains(view, "─") || strings.Contains(view, "│") {
		t.Fatalf("editor context should be borderless: %q", view)
	}
}

func TestContextInfoIsEmptyWithoutACatalog(t *testing.T) {
	state := newModelWithProjects(nil)
	state.width = 80
	if strings.TrimSpace(state.contextInfoView()) != "" {
		t.Fatalf("context line should be empty without projects: %q", state.contextInfoView())
	}
	if lipgloss.Height(state.contextInfoView()) != 2 {
		t.Fatalf("context line should always reserve two rows: %d", lipgloss.Height(state.contextInfoView()))
	}
}

func TestProjectPaneUsesQuarterTerminalWidth(t *testing.T) {
	state := newModel()
	state.width = 120
	if got, want := state.projectPanelWidth(), 30; got != want {
		t.Fatalf("project pane width = %d, want %d", got, want)
	}
	state.width = 80
	if got, want := state.projectPanelWidth(), 20; got != want {
		t.Fatalf("project pane width = %d, want %d", got, want)
	}
}

func TestShortcutFocusDoesNotResizeWorkspace(t *testing.T) {
	state, _ := send(newModel(), tea.WindowSizeMsg{Width: 120, Height: 40})
	initialHeight := lipgloss.Height(state.View())
	if initialHeight > state.height {
		t.Fatalf("workspace exceeds terminal height: %d > %d", initialHeight, state.height)
	}
	for index := 0; index < 3; index++ {
		state, _ = press(state, tea.KeyTab)
	}
	if state.focus != focusShortcuts {
		t.Fatalf("expected shortcut focus, got %s", focusLabel(state.focus))
	}
	if got := lipgloss.Height(state.View()); got != initialHeight {
		t.Fatalf("shortcut focus changed workspace height from %d to %d", initialHeight, got)
	}
}

func TestWorkspaceFitsWhenTerminalNarrows(t *testing.T) {
	state := newMockModel()
	for _, width := range []int{120, 100, 80} {
		state, _ = send(state, tea.WindowSizeMsg{Width: width, Height: 40})
		workspace := lipgloss.JoinHorizontal(lipgloss.Top, state.projectView(), "  ", state.editorView(), "  ", state.historyView())
		if got := lipgloss.Width(workspace); got > width || lipgloss.Width(state.shortcutView()) > width {
			t.Fatalf("workspace controls exceed terminal width %d: workspace=%d footer=%d", width, got, lipgloss.Width(state.shortcutView()))
		}
		state.setResult(0, bigqueryResult())
		workspace = lipgloss.JoinHorizontal(lipgloss.Top, state.projectView(), "  ", state.editorView(), "  ", state.historyView())
		if got := lipgloss.Width(workspace); got > width {
			t.Fatalf("results expanded workspace beyond terminal width %d: rendered %d", width, got)
		}
	}
}

func TestShortcutLabelsShrinkWithTerminalWidth(t *testing.T) {
	widths := []int{160, 120, 80}
	previous := 0
	for _, width := range widths {
		length := len(tabShortcutLabel(width))
		if previous != 0 && length >= previous {
			t.Fatalf("tab shortcuts did not shrink at width %d: %d >= %d", width, length, previous)
		}
		previous = length
	}
	for _, current := range []focus{focusProjects, focusResults, focusEditor} {
		if len(contextShortcutLabel(current, 80)) >= len(contextShortcutLabel(current, 160)) {
			t.Fatalf("%s shortcuts did not shrink on narrow terminals", focusLabel(current))
		}
	}
	if contextShortcutLabel(focusHistory, 80) != focusShortcutsLabel(focusHistory) {
		t.Fatal("panels without a compact label should keep the full one")
	}
}

func TestFocusLabelsCoverEveryPanel(t *testing.T) {
	for _, current := range []focus{focusProjects, focusEditor, focusResults, focusHistory, focusShortcuts} {
		if focusLabel(current) == "" || focusLabel(current) == "UNKNOWN" {
			t.Fatalf("missing focus label for %d", current)
		}
		if focusShortcutsLabel(current) == "" {
			t.Fatalf("missing shortcut label for %s", focusLabel(current))
		}
	}
	if focusLabel(focus(42)) != "UNKNOWN" {
		t.Fatalf("unexpected label for an unknown focus: %q", focusLabel(focus(42)))
	}
}

func TestWorkspaceViewRendersEveryPanel(t *testing.T) {
	state, _ := send(newMockModel(), tea.WindowSizeMsg{Width: 140, Height: 40})
	view := state.View()
	for _, want := range []string{"BIGTUI", "EXPLORER", "QUERY EDITOR", "RESULTS", "RUN HISTORY", "● "} {
		if !strings.Contains(view, want) {
			t.Fatalf("workspace view is missing %q", want)
		}
	}
}

func bigqueryResult() bigquery.Result {
	return bigquery.Result{
		Columns: []string{"customer_id", "name", "segment", "order_count", "lifetime_value"},
		Rows:    []bigquery.Row{{Values: []string{"1", "Ada Lovelace", "enterprise", "2", "4001/2"}}},
	}
}
