package app

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xjrr/bigtui/internal/project"
)

func TestProjectsTreeExpandsAndSelectsDatasets(t *testing.T) {
	state := newMockModel()
	state.focus = focusProjects
	state, _ = press(state, tea.KeyRight)
	if !state.expanded[0] {
		t.Fatal("right should expand the selected project")
	}
	state, _ = press(state, tea.KeyDown)
	if state.selectedDataset < 0 || state.projects[state.active].Resources[state.selectedDataset].Kind != "dataset" {
		t.Fatalf("down should select a dataset: project=%d resource=%d", state.active, state.selectedDataset)
	}
	state, _ = press(state, tea.KeyRight)
	if !state.datasetExpanded(state.active, state.selectedDataset) {
		t.Fatal("right should expand the selected dataset")
	}
	state, _ = press(state, tea.KeyDown)
	if state.selectedChild < 0 {
		t.Fatal("down should select a table or view inside the dataset")
	}
	state, _ = press(state, tea.KeyLeft)
	if state.selectedChild != -1 || state.selectedDataset == -1 {
		t.Fatal("left from a child should return to its dataset")
	}
	state, _ = press(state, tea.KeyLeft)
	if state.datasetExpanded(state.active, state.selectedDataset) {
		t.Fatal("left from a dataset should collapse it")
	}
	state, _ = press(state, tea.KeyLeft)
	if state.selectedDataset != -1 {
		t.Fatal("left from a collapsed dataset should return to its project")
	}
	state, _ = press(state, tea.KeyLeft)
	if state.expanded[0] {
		t.Fatal("left from a project should collapse it")
	}
}

func TestVimHorizontalKeysExpandAndCollapseProjects(t *testing.T) {
	state := newMockModel()
	state.focus = focusProjects
	state, _ = typeRunes(state, 'l')
	if !state.expanded[0] {
		t.Fatal("l should expand the selected project")
	}
	state, _ = typeRunes(state, 'h')
	if state.expanded[0] {
		t.Fatal("h should collapse the selected project")
	}
}

func TestVimVerticalKeysMoveThroughProjects(t *testing.T) {
	state := newMockModel()
	state.focus = focusProjects
	state, _ = typeRunes(state, 'j')
	if state.active != 1 {
		t.Fatalf("j should move to the next project, got %d", state.active)
	}
	state, _ = typeRunes(state, 'k')
	if state.active != 0 {
		t.Fatalf("k should move back to the previous project, got %d", state.active)
	}
}

func TestProjectSelectionStopsAtCatalogBounds(t *testing.T) {
	state := newMockModel()
	state.focus = focusProjects
	for index := 0; index < 5; index++ {
		state.moveProjectSelection(-1)
	}
	if state.active != 0 {
		t.Fatalf("selection moved above the first project: %d", state.active)
	}
	for index := 0; index < 5; index++ {
		state.moveProjectSelection(1)
	}
	if state.active != len(state.projects)-1 {
		t.Fatalf("selection moved past the last project: %d", state.active)
	}
}

func TestProjectSelectionContinuesPastExpandedChildren(t *testing.T) {
	state := newModelWithProjects([]project.Project{
		{ID: "project-1", Resources: []project.Resource{
			{Name: "dataset-1", Kind: "dataset", Children: []project.Resource{{Name: "table-1", Kind: "table"}}},
			{Name: "dataset-2", Kind: "dataset", Children: []project.Resource{{Name: "table-2", Kind: "table"}}},
		}},
		{ID: "project-2", Resources: []project.Resource{{Name: "dataset-3", Kind: "dataset"}}},
	})
	state.focus = focusProjects
	state.expanded[0] = true
	state.expandedDataset[datasetKey(0, 0)] = true
	state.expandedDataset[datasetKey(0, 1)] = true
	state.selectedDataset = 0
	state.selectedChild = 0

	state.moveProjectSelection(1)
	if state.selectedDataset != 1 || state.selectedChild != -1 {
		t.Fatalf("down from the last child should advance to the next dataset: dataset=%d child=%d", state.selectedDataset, state.selectedChild)
	}
	state.moveProjectSelection(1)
	if state.selectedDataset != 1 || state.selectedChild != 0 {
		t.Fatalf("down should enter the next expanded dataset: dataset=%d child=%d", state.selectedDataset, state.selectedChild)
	}
	state.moveProjectSelection(1)
	if state.active != 1 || state.selectedDataset != -1 || state.selectedChild != -1 {
		t.Fatalf("down from the last expanded dataset child should advance to the next project: project=%d dataset=%d child=%d", state.active, state.selectedDataset, state.selectedChild)
	}
}

func TestProjectPaneScrollsAtViewportEdges(t *testing.T) {
	children := make([]project.Resource, 20)
	for index := range children {
		children[index] = project.Resource{Name: fmt.Sprintf("table_%02d", index), Kind: "table", DetailsLoaded: true}
	}
	state := newModelWithProjects([]project.Project{{ID: "project-1", Resources: []project.Resource{{Name: "dataset-1", Kind: "dataset", Children: children}}}})
	state.width = 80
	state.height = 20
	state.focus = focusProjects
	state.expanded[0] = true
	state.selectedDataset = 0
	state.expandedDataset[datasetKey(0, 0)] = true
	state.selectedChild = 0
	state.updateProjectScroll()

	for index := 0; index < 10; index++ {
		state.moveProjectSelection(1)
	}
	if state.projectScroll == 0 {
		t.Fatal("moving down through tables should scroll the project pane")
	}
	scrollAtBottom := state.projectScroll
	state.moveProjectSelection(-1)
	if state.projectScroll != scrollAtBottom {
		t.Fatalf("moving up within the bottom viewport should keep the pane down: got %d, want %d", state.projectScroll, scrollAtBottom)
	}
	for index := 0; index < 20; index++ {
		state.moveProjectSelection(-1)
	}
	if state.projectScroll != 0 {
		t.Fatalf("moving to the first table should scroll the pane to the top: got %d", state.projectScroll)
	}
}

func TestProjectRowsRenderTheExpandedTree(t *testing.T) {
	state := newMockModel()
	state.width, state.height = 120, 40
	state.expanded[0] = true
	state.selectedDataset = 0
	state.expandedDataset[datasetKey(0, 0)] = true
	rows := state.projectRows()
	rendered := []string{}
	for _, row := range rows {
		rendered = append(rendered, row.text)
	}
	joined := strings.Join(rendered, "\n")
	for _, want := range []string{"Sandbox Analytics", "events", "customers", "Sandbox Reporting"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("explorer rows are missing %q: %q", want, joined)
		}
	}
	if selectedRowIndex(rows) == 0 {
		t.Fatal("the selected dataset row should be marked as selected")
	}
}

func TestProjectRowsShowLoadingPlaceholders(t *testing.T) {
	state := newModelWithLoader([]project.Project{{ID: "project-1", Resources: []project.Resource{{Name: "dataset-1", Kind: "dataset"}}}}, &catalogLoaderStub{})
	state.width, state.height = 120, 40
	state.expanded[0] = true
	state.projectLoading[0] = true
	if !strings.Contains(renderRows(state.projectRows()), "Loading datasets...") {
		t.Fatal("an in-flight dataset request should render a placeholder")
	}
	state.projectLoading[0] = false
	state.expandedDataset[datasetKey(0, 0)] = true
	state.datasetLoading[datasetKey(0, 0)] = true
	if !strings.Contains(renderRows(state.projectRows()), "Loading tables...") {
		t.Fatal("an in-flight table request should render a placeholder")
	}
}

func TestEmptyCatalogRendersGuidance(t *testing.T) {
	state := newModelWithProjects(nil)
	state.width, state.height = 100, 30
	if !strings.Contains(state.projectView(), "No projects connected.") {
		t.Fatalf("empty explorer should explain the state: %q", state.projectView())
	}
}

func TestCtrlEInsertsSelectedDatasetAtEditorCursor(t *testing.T) {
	state := newMockModel()
	state.focus = focusProjects
	state.selectedDataset = 0
	state.selectedChild = -1
	state.tabs[0].editor.SetValue("-- query\n")
	state.tabs[0].editor.CursorEnd()
	state, _ = press(state, tea.KeyCtrlE)
	want := "-- query\nSELECT * FROM `sandbox-analytics.events`"
	if state.tabs[0].editor.Value() != want {
		t.Fatalf("dataset query insertion = %q, want %q", state.tabs[0].editor.Value(), want)
	}
	if state.focus != focusEditor {
		t.Fatalf("expected focus to return to editor, got %s", focusLabel(state.focus))
	}
}

func TestCtrlEInsertsSelectedTableAtEditorCursor(t *testing.T) {
	state := newMockModel()
	state.focus = focusProjects
	state.selectedDataset = 0
	state.selectedChild = 0
	state.tabs[0].editor.SetValue("SELECT 1; ")
	state.tabs[0].editor.CursorEnd()
	state, _ = press(state, tea.KeyCtrlE)
	want := "SELECT 1; SELECT * FROM `sandbox-analytics.events.customers`"
	if state.tabs[0].editor.Value() != want {
		t.Fatalf("table query insertion = %q, want %q", state.tabs[0].editor.Value(), want)
	}
}

func TestCtrlEWithoutASelectedDatasetKeepsTheQuery(t *testing.T) {
	state := newMockModel()
	state.focus = focusProjects
	state.tabs[0].editor.SetValue("SELECT 1")
	state, _ = press(state, tea.KeyCtrlE)
	if state.tabs[0].editor.Value() != "SELECT 1" || state.focus != focusProjects {
		t.Fatalf("insertion without a dataset changed the workspace: query=%q focus=%s", state.tabs[0].editor.Value(), focusLabel(state.focus))
	}
}

func renderRows(rows []projectRow) string {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, row.text)
	}
	return strings.Join(lines, "\n")
}
