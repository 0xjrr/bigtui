package app

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xjrr/bigtui/internal/project"
)

func TestEnterOpensResourceInfoModal(t *testing.T) {
	state := newMockModel()
	state.focus = focusProjects
	state, _ = press(state, tea.KeyEnter)
	if !state.showInfo || !strings.Contains(state.infoView(), "Sandbox Analytics") {
		t.Fatal("enter should open project information")
	}
	state, _ = press(state, tea.KeyEscape)
	if state.showInfo {
		t.Fatal("escape should close resource information")
	}
	state.expanded[0] = true
	state.selectedDataset = 0
	state.expandedDataset[datasetKey(0, 0)] = true
	state.selectedChild = 1
	state, _ = press(state, tea.KeyEnter)
	if !state.showInfo || !strings.Contains(state.infoView(), "customer_order_totals") {
		t.Fatal("enter should show selected child information")
	}
	state.selectedChild = 0
	if !strings.Contains(infoText(state), "Ada Lovelace") || !strings.Contains(infoText(state), "Preview") {
		t.Fatal("table information should include its data preview")
	}
	state.selectedChild = 2
	if !strings.Contains(infoText(state), "gs://partner-feed/events/*.parquet") {
		t.Fatal("external table information should include its source")
	}
}

func TestDatasetInfoSummarizesChildren(t *testing.T) {
	state := newMockModel()
	state.selectedDataset = 0
	state.selectedChild = -1
	text := infoText(state)
	if !strings.Contains(text, "DATASET") || !strings.Contains(text, "Children  3 tables/views") {
		t.Fatalf("dataset info should summarize its children: %q", text)
	}
}

func TestPendingResourceDetailsAreAnnounced(t *testing.T) {
	state := newModelWithProjects([]project.Project{{ID: "project-1", Resources: []project.Resource{
		{Name: "dataset-1", Kind: "dataset", Children: []project.Resource{{Name: "table-1", Kind: "table"}}},
	}}})
	state.selectedDataset = 0
	state.selectedChild = 0
	text := infoText(state)
	if !strings.Contains(text, "Details  Loading...") || strings.Contains(text, "Table info") {
		t.Fatalf("unloaded resources should only show a loading hint: %q", text)
	}
}

func TestResourceInfoListsMetadataSections(t *testing.T) {
	resource := project.Resource{
		Name: "orders", Kind: "table", DetailsLoaded: true, ID: "project:dataset.orders",
		Description: "Order facts", Location: "US", LegacySQL: false,
		Labels: map[string]string{"team": "data"}, NumRows: 10, NumBytes: 2048, LongTermBytes: 1024,
		PartitionType: "DAY", PartitionField: "ordered_at", RequirePartitionFilter: true,
		Clustering: []string{"customer_id"},
	}
	lines := strings.Join(resourceInfoLines(resource), "\n")
	for _, want := range []string{"Table info", "Expiration     NEVER", "Legacy SQL     false", "team=data", "Storage info", "Number of rows          10", "Partitioning", "Type             DAY", "Require filter   true", "Clustering", "customer_id"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("resource info is missing %q: %q", want, lines)
		}
	}
}

func TestViewResourceInfoSkipsStorageSection(t *testing.T) {
	lines := strings.Join(resourceInfoLines(project.Resource{Name: "monthly", Kind: "view", DetailsLoaded: true}), "\n")
	if strings.Contains(lines, "Storage info") {
		t.Fatalf("views should not report storage metrics: %q", lines)
	}
}

func TestViewQueryRemainsOneStyledBlock(t *testing.T) {
	state := newMockModel()
	state.selectedDataset = 0
	state.selectedChild = 1
	state.showInfo = true
	joined := infoText(state)
	compact := strings.ReplaceAll(strings.ReplaceAll(joined, " ", ""), "\n", "")
	if !strings.Contains(compact, "SELECTcustomer_id") || !strings.Contains(compact, "GROUPBYcustomer_id") {
		t.Fatalf("view query block was not preserved: %q", joined)
	}
}

func TestInfoModalFillsTerminal(t *testing.T) {
	state := newMockModel()
	state.width, state.height = 100, 30
	state.focus = focusProjects
	state.showInfo = true
	view := state.infoView()
	if lipgloss.Width(view) < 90 || lipgloss.Height(view) < 26 {
		t.Fatalf("modal should fill terminal: %dx%d", lipgloss.Width(view), lipgloss.Height(view))
	}
}

func TestInfoModalUsesFixedViewportForLargeContent(t *testing.T) {
	state := newMockModel()
	state.width, state.height = 80, 20
	state.selectedDataset = 0
	state.selectedChild = 0
	state.showInfo = true
	state.projects[0].Resources[0].Children[0].Columns = []string{"id", "description"}
	for index := 0; index < 40; index++ {
		state.projects[0].Resources[0].Children[0].Preview = append(state.projects[0].Resources[0].Children[0].Preview, []string{fmt.Sprint(index), strings.Repeat("large-value ", 8)})
	}
	view := state.infoView()
	if lipgloss.Width(view) > state.width || lipgloss.Height(view) > state.height {
		t.Fatalf("info modal exceeded terminal: %dx%d in %dx%d", lipgloss.Width(view), lipgloss.Height(view), state.width, state.height)
	}
	state.infoScroll = 100
	scrolled := state.infoView()
	if lipgloss.Width(scrolled) > state.width || lipgloss.Height(scrolled) > state.height {
		t.Fatalf("scrolled info modal exceeded terminal: %dx%d in %dx%d", lipgloss.Width(scrolled), lipgloss.Height(scrolled), state.width, state.height)
	}
}

func TestInfoModalScrollKeysMoveTheViewport(t *testing.T) {
	state := newMockModel()
	state.width, state.height = 80, 20
	state.showInfo = true
	state, _ = press(state, tea.KeyDown)
	if state.infoScroll != 1 {
		t.Fatalf("down should scroll one line: %d", state.infoScroll)
	}
	state, _ = press(state, tea.KeyUp)
	state, _ = press(state, tea.KeyUp)
	if state.infoScroll != 0 {
		t.Fatalf("scrolling above the first line should clamp to zero: %d", state.infoScroll)
	}
	state, _ = send(state, tea.KeyMsg{Type: tea.KeyPgDown})
	if state.infoScroll != state.pageStep() {
		t.Fatalf("pgdown should scroll one page: %d", state.infoScroll)
	}
	state, _ = send(state, tea.KeyMsg{Type: tea.KeyPgUp})
	if state.infoScroll != 0 {
		t.Fatalf("pgup should scroll back one page: %d", state.infoScroll)
	}
}

func TestHelpModalShowsCompleteKeymapAndClosesWithoutQuitting(t *testing.T) {
	state := newModel()
	state.width, state.height = 100, 24
	state.focus = focusProjects
	state, command := typeRunes(state, '?')
	if command != nil || !state.showHelp {
		t.Fatal("? should open the help modal without a command")
	}
	view := state.helpView()
	for _, want := range []string{"GENERAL", "EXPLORER", "ctrl+e", "QUERY EDITOR", "COMPLETION", "RESULTS", "RUN HISTORY", "SEARCH", "RESOURCE INFO", "HELP", "Esc/?/q"} {
		found := strings.Contains(view, want)
		for scroll := 1; !found && scroll <= 100; scroll++ {
			state.helpScroll = scroll
			found = strings.Contains(state.helpView(), want)
		}
		if !found {
			t.Fatalf("help modal is missing %q", want)
		}
	}
	state.helpScroll = 0
	view = state.helpView()
	if lipgloss.Width(view) > state.width || lipgloss.Height(view) > state.height {
		t.Fatalf("help modal exceeded terminal: %dx%d in %dx%d", lipgloss.Width(view), lipgloss.Height(view), state.width, state.height)
	}
	state, command = typeRunes(state, 'q')
	if command != nil || state.showHelp {
		t.Fatal("q should close help without quitting the TUI")
	}
}

func TestHelpModalSwallowsWorkspaceShortcuts(t *testing.T) {
	state := newModel()
	state.width, state.height = 100, 24
	state.focus = focusProjects
	state, _ = typeRunes(state, '?')
	state, command := press(state, tea.KeyCtrlN)
	if command != nil || len(state.tabs) != 1 {
		t.Fatalf("help should swallow workspace shortcuts: tabs=%d", len(state.tabs))
	}
	state, _ = send(state, tea.KeyMsg{Type: tea.KeyPgDown})
	if state.helpScroll != state.pageStep() {
		t.Fatalf("pgdown should scroll the help modal: %d", state.helpScroll)
	}
}

func infoText(state model) string {
	return strings.Join(state.infoLines(), "\n")
}
