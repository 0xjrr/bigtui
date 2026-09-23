package app

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/0xjrr/bigtui/internal/project"
)

func TestCatalogLoadsDatasetsAndTablesOnExpansion(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := newModelWithLoader(nil, loader)
	if state.Init() == nil {
		t.Fatal("initial catalog load should return a command")
	}
	state, _ = send(state, state.loadProjects()())
	if len(state.projects) != 1 || len(state.projects[0].Resources) != 0 {
		t.Fatalf("initial catalog load should not enumerate datasets: %#v", state.projects)
	}
	state.focus = focusProjects
	state, command := press(state, tea.KeyRight)
	if command == nil || loader.datasetCalls != 0 {
		t.Fatalf("project expansion should schedule dataset loading: command=%v calls=%d", command != nil, loader.datasetCalls)
	}
	state = runCommand(t, state, command)
	if loader.datasetCalls != 1 || loader.tableCalls != 0 || len(state.projects[0].Resources) != 1 {
		t.Fatalf("unexpected dataset loading state: datasets=%d tables=%d resources=%#v", loader.datasetCalls, loader.tableCalls, state.projects[0].Resources)
	}
	if loader.lastIncludeHidden {
		t.Fatal("hidden datasets should be excluded by default")
	}
	state, _ = press(state, tea.KeyDown)
	state, command = press(state, tea.KeyRight)
	if command == nil {
		t.Fatal("dataset expansion should schedule table loading")
	}
	state = runCommand(t, state, command)
	if loader.tableCalls != 1 || len(state.projects[0].Resources[0].Children) != 1 {
		t.Fatalf("unexpected table loading state: calls=%d children=%#v", loader.tableCalls, state.projects[0].Resources[0].Children)
	}
}

func TestExpandingAnAlreadyLoadedDatasetDoesNotReload(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := newModelWithLoader([]project.Project{{ID: "project-1", Resources: []project.Resource{{Name: "dataset-1", Kind: "dataset"}}}}, loader)
	state.focus = focusProjects
	state.datasetsLoaded[0] = true
	state.selectedDataset = 0
	state.tablesLoaded[datasetKey(0, 0)] = true
	if command := state.expandProject(); command != nil {
		t.Fatal("a cached dataset should not schedule another table load")
	}
	if loader.tableCalls != 0 {
		t.Fatalf("unexpected table calls: %d", loader.tableCalls)
	}
}

func TestCatalogLoadFailuresSurfaceInTheStatusLine(t *testing.T) {
	state := newModelWithLoader([]project.Project{{ID: "project-1"}}, &catalogLoaderStub{})
	state, _ = send(state, projectsLoaded{err: errors.New("denied")})
	if state.status != "Project loading failed: denied" {
		t.Fatalf("unexpected project failure status: %q", state.status)
	}
	state, _ = send(state, datasetsLoaded{projectIndex: 0, err: errors.New("no access")})
	if state.status != "Dataset loading failed: no access" || state.projectLoading[0] {
		t.Fatalf("unexpected dataset failure state: status=%q loading=%v", state.status, state.projectLoading[0])
	}
	state, _ = send(state, tablesLoaded{projectIndex: 0, datasetIndex: 0, err: errors.New("missing")})
	if state.status != "Table loading failed: missing" || state.datasetLoading[datasetKey(0, 0)] {
		t.Fatalf("unexpected table failure state: status=%q loading=%v", state.status, state.datasetLoading[datasetKey(0, 0)])
	}
	state.resourceLoading = true
	state, _ = send(state, resourceLoaded{err: errors.New("gone")})
	if state.status != "Resource details failed: gone" || state.resourceLoading {
		t.Fatalf("unexpected resource failure state: status=%q loading=%v", state.status, state.resourceLoading)
	}
}

func TestStaleHiddenDatasetResponsesAreDropped(t *testing.T) {
	state := newModelWithLoader([]project.Project{{ID: "project-1"}}, &catalogLoaderStub{})
	state, _ = send(state, datasetsLoaded{projectIndex: 0, includeHidden: true, resources: []project.Resource{{Name: "hidden", Kind: "dataset"}}})
	if len(state.projects[0].Resources) != 0 {
		t.Fatalf("responses for a different visibility mode should be ignored: %#v", state.projects[0].Resources)
	}
}

func TestCtrlHTogglesHiddenDatasets(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := newModelWithLoader([]project.Project{{ID: "project-1"}}, loader)
	state.focus = focusProjects
	state.expanded[0] = true
	state, command := press(state, tea.KeyCtrlH)
	if !state.showHiddenDatasets || command == nil {
		t.Fatal("Ctrl+H should enable hidden datasets and reload expanded projects")
	}
	state = runCommand(t, state, command)
	if !loader.lastIncludeHidden {
		t.Fatal("enabled hidden-dataset mode should request hidden datasets")
	}
}

func TestCtrlHIsUnavailableWithoutALoader(t *testing.T) {
	state := newMockModel()
	state.focus = focusProjects
	state, command := press(state, tea.KeyCtrlH)
	if command != nil || state.showHiddenDatasets {
		t.Fatal("mock mode should not toggle hidden datasets")
	}
	if state.status != "Hidden dataset filtering is unavailable in mock mode." {
		t.Fatalf("unexpected status: %q", state.status)
	}
}

func TestEnterLoadsResourceDetailsOnce(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := newModelWithLoader([]project.Project{{ID: "project-1", Resources: []project.Resource{
		{Name: "dataset-1", Kind: "dataset", Children: []project.Resource{{Name: "table-1", Kind: "table"}}},
	}}}, loader)
	state.focus = focusProjects
	state.expanded[0] = true
	state.selectedDataset = 0
	state.selectedChild = 0
	state, command := press(state, tea.KeyEnter)
	if command == nil || !state.resourceLoading || !strings.HasPrefix(state.status, "Loading details for") {
		t.Fatalf("enter should request resource details: command=%v loading=%v status=%q", command != nil, state.resourceLoading, state.status)
	}
	state = runCommand(t, state, command)
	if loader.resourceCalls != 1 || state.resourceLoading {
		t.Fatalf("unexpected resource load state: calls=%d loading=%v", loader.resourceCalls, state.resourceLoading)
	}
	if !state.projects[0].Resources[0].Children[0].DetailsLoaded {
		t.Fatal("loaded details should be stored on the child resource")
	}
	state.showInfo = false
	state, command = press(state, tea.KeyEnter)
	if command != nil {
		t.Fatal("already loaded details should not be requested again")
	}
}
