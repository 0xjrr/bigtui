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
	"github.com/xjrr/bigtui/internal/completion"
	"github.com/xjrr/bigtui/internal/project"
)

type clientFunc func(context.Context, string, string) (bigquery.Result, error)

func (f clientFunc) Query(ctx context.Context, projectID, sql string) (bigquery.Result, error) {
	return f(ctx, projectID, sql)
}

type catalogLoaderStub struct {
	datasetCalls      int
	lastIncludeHidden bool
	tableCalls        int
}

func (s *catalogLoaderStub) Load(context.Context) ([]project.Project, error) {
	return []project.Project{{ID: "project-1", Name: "Project 1"}}, nil
}

func (s *catalogLoaderStub) LoadDatasets(_ context.Context, _ string, includeHidden bool) ([]project.Resource, error) {
	s.datasetCalls++
	s.lastIncludeHidden = includeHidden
	return []project.Resource{{Name: "dataset-1", Kind: "dataset"}}, nil
}

func (s *catalogLoaderStub) LoadTables(context.Context, string, string) ([]project.Resource, error) {
	s.tableCalls++
	return []project.Resource{{Name: "table-1", Kind: "table"}}, nil
}

func (s *catalogLoaderStub) LoadResource(context.Context, string, string, string) (project.Resource, error) {
	return project.Resource{}, nil
}

func TestCatalogLoadsDatasetsAndTablesOnExpansion(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := initialModelWithProjectsAndLoader(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), nil, loader)
	command := state.Init()
	if command == nil {
		t.Fatal("initial catalog load should return a command")
	}
	message := state.loadProjects()()
	updatedModel, _ := state.Update(message)
	state = updatedModel.(model)
	if len(state.projects) != 1 || len(state.projects[0].Resources) != 0 {
		t.Fatalf("initial catalog load should not enumerate datasets: %#v", state.projects)
	}
	state.focus = focusProjects
	updatedModel, command = state.Update(tea.KeyMsg{Type: tea.KeyRight})
	state = updatedModel.(model)
	if command == nil || loader.datasetCalls != 0 {
		t.Fatalf("project expansion should schedule dataset loading: command=%v calls=%d", command != nil, loader.datasetCalls)
	}
	updatedModel, _ = state.Update(command())
	state = updatedModel.(model)
	if loader.datasetCalls != 1 || loader.tableCalls != 0 || len(state.projects[0].Resources) != 1 {
		t.Fatalf("unexpected dataset loading state: datasets=%d tables=%d resources=%#v", loader.datasetCalls, loader.tableCalls, state.projects[0].Resources)
	}
	if loader.lastIncludeHidden {
		t.Fatal("hidden datasets should be excluded by default")
	}
	updatedModel, command = state.Update(tea.KeyMsg{Type: tea.KeyDown})
	state = updatedModel.(model)
	updatedModel, command = state.Update(tea.KeyMsg{Type: tea.KeyRight})
	state = updatedModel.(model)
	if command == nil {
		t.Fatal("dataset expansion should schedule table loading")
	}
	updatedModel, _ = state.Update(command())
	state = updatedModel.(model)
	if loader.tableCalls != 1 || len(state.projects[0].Resources[0].Children) != 1 {
		t.Fatalf("unexpected table loading state: calls=%d children=%#v", loader.tableCalls, state.projects[0].Resources[0].Children)
	}
}

func TestAutocompleteLoadsDatasetsOnlyForResourceContext(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := initialModelWithProjectsAndLoader(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), []project.Project{{ID: "project-1"}}, loader)
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("SELECT cu")
	state.tabs[0].editor.CursorEnd()
	if command := state.refreshCompletion(); command != nil {
		t.Fatal("typing a non-resource SQL prefix should not load datasets")
	}
	if loader.datasetCalls != 0 {
		t.Fatalf("unexpected dataset request count: %d", loader.datasetCalls)
	}
	state.tabs[0].editor.SetValue("SELECT * FROM d")
	state.tabs[0].editor.CursorEnd()
	command := state.refreshCompletion()
	if command == nil {
		t.Fatal("resource completion should request datasets on demand")
	}
	updated, _ := state.Update(command())
	state = updated.(model)
	if loader.datasetCalls != 1 || loader.lastIncludeHidden {
		t.Fatalf("unexpected default dataset request: calls=%d includeHidden=%v", loader.datasetCalls, loader.lastIncludeHidden)
	}
}

func TestAutocompleteLoadsTablesForQualifiedDataset(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := initialModelWithProjectsAndLoader(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), []project.Project{{ID: "project-1", Resources: []project.Resource{{Name: "dataset-1", Kind: "dataset"}}}}, loader)
	state.focus = focusEditor
	state.datasetsLoaded[0] = true
	state.tabs[0].editor.SetValue("SELECT * FROM `project-1`.`dataset-1`.")
	state.tabs[0].editor.CursorEnd()
	command := state.refreshCompletion()
	if command == nil {
		t.Fatal("qualified dataset completion should request tables on demand")
	}
	updated, _ := state.Update(command())
	state = updated.(model)
	if loader.tableCalls != 1 {
		t.Fatalf("expected one table request, got %d", loader.tableCalls)
	}
	if !state.completionOpen || len(state.completionItems) == 0 || state.completionItems[0].Label != "table-1" {
		t.Fatalf("expected table completion after loading: %#v", state.completionItems)
	}
}

func TestAutocompleteDatasetLoadingHonorsHiddenDatasetToggle(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := initialModelWithProjectsAndLoader(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), []project.Project{{ID: "project-1"}}, loader)
	state.focus = focusEditor
	state.showHiddenDatasets = true
	state.tabs[0].editor.SetValue("SELECT * FROM d")
	state.tabs[0].editor.CursorEnd()
	command := state.refreshCompletion()
	if command == nil {
		t.Fatal("resource completion should request datasets")
	}
	updated, _ := state.Update(command())
	state = updated.(model)
	if !loader.lastIncludeHidden {
		t.Fatal("autocomplete should include hidden datasets when Ctrl+H mode is active")
	}
}

func TestCtrlHTogglesHiddenDatasets(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := initialModelWithProjectsAndLoader(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), []project.Project{{ID: "project-1"}}, loader)
	state.focus = focusProjects
	state.expanded[0] = true
	updated, command := state.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	state = updated.(model)
	if !state.showHiddenDatasets || command == nil {
		t.Fatal("Ctrl+H should enable hidden datasets and reload expanded projects")
	}
	updated, _ = state.Update(command())
	state = updated.(model)
	if !loader.lastIncludeHidden {
		t.Fatal("enabled hidden-dataset mode should request hidden datasets")
	}
}

func TestTypingOpensCompletionAndEnterInserts(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("SELECT * FROM cus")
	state.tabs[0].editor.CursorEnd()
	if state.completionOpen {
		t.Fatal("completion should not be open before typing")
	}
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	state = updated.(model)
	if !state.completionOpen || len(state.completionItems) == 0 {
		t.Fatalf("typing a matching prefix should open completion automatically: open=%v items=%d", state.completionOpen, len(state.completionItems))
	}
	found := false
	for state.completionItems[state.completionCursor].Label != "customers" {
		if state.completionCursor >= len(state.completionItems)-1 {
			break
		}
		updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyDown})
		state = updated.(model)
	}
	for _, item := range state.completionItems {
		if item.Label == "customers" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a customers table suggestion: %#v", state.completionItems)
	}
	if state.completionItems[state.completionCursor].Label != "customers" {
		t.Fatalf("down navigation should reach the customers suggestion: %#v", state.completionItems)
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state = updated.(model)
	if state.completionOpen {
		t.Fatal("enter should close the completion popup")
	}
	if !strings.HasSuffix(state.tabs[0].editor.Value(), "customers") {
		t.Fatalf("enter should insert the selected completion in place of the typed prefix: %q", state.tabs[0].editor.Value())
	}
}

func TestSFAcceptanceInsertsSelectFromSnippet(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("s")
	state.tabs[0].editor.CursorEnd()
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	state = updated.(model)
	if !state.completionOpen || len(state.completionItems) == 0 || state.completionItems[0].InsertText != "SELECT * FROM `" {
		t.Fatalf("expected sf snippet suggestion: %#v", state.completionItems)
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state = updated.(model)
	if got := state.tabs[0].editor.Value(); got != "SELECT * FROM `" {
		t.Fatalf("sf acceptance = %q, want %q", got, "SELECT * FROM `")
	}
}

func TestCompletionAcceptanceAddsQualifiedNameDelimiters(t *testing.T) {
	tests := []struct {
		name  string
		value string
		item  completion.Item
		want  string
	}{
		{name: "project dot", value: "FROM demo", item: completion.Item{Label: "demo-project", Detail: "project", InsertText: "demo-project"}, want: "FROM demo-project."},
		{name: "quoted dataset dot", value: "FROM `demo-project`.`cust", item: completion.Item{Label: "customers", Detail: "dataset · demo-project", InsertText: "customers"}, want: "FROM `demo-project`.`customers."},
		{name: "quoted view close", value: "FROM `demo-project.customers.customer_v", item: completion.Item{Label: "customer_view", Detail: "view · customers", InsertText: "customer_view"}, want: "FROM `demo-project.customers.customer_view`"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
				return bigquery.Result{}, nil
			}))
			state.tabs[0].editor.SetValue(testCase.value)
			state.tabs[0].editor.CursorEnd()
			state.completionItems = []completion.Item{testCase.item}
			state.completionOpen = true
			state.acceptCompletion()
			if got := state.tabs[0].editor.Value(); got != testCase.want {
				t.Fatalf("accepted completion = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestCompletionAcceptanceRefreshesNextQualifiedLevel(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.tabs[0].editor.SetValue("FROM sandbox-")
	state.tabs[0].editor.CursorEnd()
	state.completionItems = []completion.Item{{Label: "sandbox-analytics", Detail: "project", InsertText: "sandbox-analytics"}}
	state.completionOpen = true
	state.acceptCompletion()
	if got := state.tabs[0].editor.Value(); got != "FROM sandbox-analytics." {
		t.Fatalf("accepted project = %q, want %q", got, "FROM sandbox-analytics.")
	}
	if !state.completionOpen || len(state.completionItems) == 0 {
		t.Fatalf("expected dataset suggestions immediately after project dot: %#v", state.completionItems)
	}
	for _, item := range state.completionItems {
		if item.Detail == "project" {
			t.Fatalf("project suggestions should not reappear after project dot: %#v", state.completionItems)
		}
	}
}

func TestCompletionClosesWhenPrefixBecomesEmpty(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("SELECT * FROM cust")
	state.tabs[0].editor.CursorEnd()
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	state = updated.(model)
	if !state.completionOpen {
		t.Fatal("expected completion open after typing a matching prefix")
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	state = updated.(model)
	if state.completionOpen {
		t.Fatal("completion should close once the prefix becomes empty")
	}
}

func TestEscClosesCompletionPopupWithoutChangingQuery(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("SELECT * FROM cust")
	state.tabs[0].editor.CursorEnd()
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	state = updated.(model)
	if !state.completionOpen {
		t.Fatal("expected completion popup to open")
	}
	value := state.tabs[0].editor.Value()
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEscape})
	state = updated.(model)
	if state.completionOpen || len(state.completionItems) != 0 {
		t.Fatal("esc should close the completion popup")
	}
	if state.tabs[0].editor.Value() != value {
		t.Fatalf("esc should not modify the query: got %q, want %q", state.tabs[0].editor.Value(), value)
	}
}

func TestOverlaySplicesPopupAtGivenPosition(t *testing.T) {
	base := "AAAAAAAAAA\nBBBBBBBBBB\nCCCCCCCCCC"
	got := overlay(base, "XY\nZW", 1, 3)
	want := "AAAAAAAAAA\nBBBXYBBBBB\nCCCZWCCCCC"
	if got != want {
		t.Fatalf("overlay placed popup incorrectly:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestOverlayPadsShortLinesAndIgnoresOutOfRangeRows(t *testing.T) {
	got := overlay("A\nB", "XYZ", 0, 3)
	if got != "A  XYZ\nB" {
		t.Fatalf("overlay should pad short lines before the popup column: %q", got)
	}
	got = overlay("A", "XYZ", 5, 0)
	if got != "A" {
		t.Fatalf("overlay should ignore rows outside the base content: %q", got)
	}
}

func TestCompletionPopupOverlaysNearCursorWithoutGrowingTheView(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.width, state.height = 120, 40
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("SELECT * FROM cust")
	state.tabs[0].editor.CursorEnd()
	baseView := state.View()
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	state = updated.(model)
	if !state.completionOpen {
		t.Fatal("expected completion popup to open")
	}
	if strings.Contains(state.editorView(), "customers") {
		t.Fatal("completion suggestions should not be appended inline in the editor panel")
	}
	overlaid := state.View()
	if !contains(overlaid, "customers") {
		t.Fatalf("overlaid view should contain the completion popup: %q", overlaid)
	}
	if lipgloss.Height(overlaid) != lipgloss.Height(baseView) {
		t.Fatalf("overlay should not change the rendered view height: got %d, want %d", lipgloss.Height(overlaid), lipgloss.Height(baseView))
	}
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

func TestCtrlSOpensQualifiedResourceSearch(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	state = updated.(model)
	if !state.searchOpen || len(state.searchResults) == 0 {
		t.Fatal("ctrl+s should open a search with resources")
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	state = updated.(model)
	if !contains(state.searchView(), "customers") || !contains(state.searchView(), "sandbox-analytics") {
		t.Fatalf("search should show qualified resource locations: %q", state.searchView())
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state = updated.(model)
	if state.searchOpen || state.focus != focusProjects {
		t.Fatal("enter should select a search result and return to Projects focus")
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

func TestCtrlEInsertsSelectedDatasetAtEditorCursor(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.focus = focusProjects
	state.selectedDataset = 0
	state.selectedChild = -1
	state.tabs[0].editor.SetValue("-- query\n")
	state.tabs[0].editor.CursorEnd()
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	state = updated.(model)
	want := "-- query\nSELECT * FROM `sandbox-analytics.events`"
	if state.tabs[0].editor.Value() != want {
		t.Fatalf("dataset query insertion = %q, want %q", state.tabs[0].editor.Value(), want)
	}
	if state.focus != focusEditor {
		t.Fatalf("expected focus to return to editor, got %s", focusLabel(state.focus))
	}
}

func TestCtrlEInsertsSelectedTableAtEditorCursor(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	state.focus = focusProjects
	state.selectedDataset = 0
	state.selectedChild = 0
	state.tabs[0].editor.SetValue("SELECT 1; ")
	state.tabs[0].editor.CursorEnd()
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	state = updated.(model)
	want := "SELECT 1; SELECT * FROM `sandbox-analytics.events.customers`"
	if state.tabs[0].editor.Value() != want {
		t.Fatalf("table query insertion = %q, want %q", state.tabs[0].editor.Value(), want)
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

func TestInfoModalUsesFixedViewportForLargeContent(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
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

func TestProjectContextShowsFullNameWithoutBorder(t *testing.T) {
	state := initialModelWithProjects(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), []project.Project{{ID: "project-id", Name: "project-name", Resources: []project.Resource{{Name: "dataset-name", Kind: "dataset", Children: []project.Resource{{Name: "a-very-long-table-name", Kind: "table"}}}}}})
	state.width = 80
	state.active = 0
	state.selectedDataset = 0
	state.selectedChild = 0
	view := state.contextInfoView()
	compact := strings.ReplaceAll(strings.ReplaceAll(view, " ", ""), "\n", "")
	if !contains(compact, "project-id.dataset-name.a-very-long-table-name") {
		t.Fatalf("selected resource path is missing: %q", view)
	}
	if strings.Contains(view, "...") || strings.Contains(view, "─") || strings.Contains(view, "│") {
		t.Fatalf("selected resource view should be untruncated and unbordered: %q", view)
	}
}

func TestEditorContextShowsSelectedCompletionDescription(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	state.width = 80
	state.focus = focusEditor
	state.completionOpen = true
	state.completionItems = []completion.Item{{Label: "S2_CELLIDFROMPOINT", Detail: "Gets the S2 cell ID covering a point GEOGRAPHY value."}}
	view := state.contextInfoView()
	if !contains(view, "S2_CELLIDFROMPOINT") || !contains(view, "Gets the S2 cell ID") {
		t.Fatalf("editor context should show selected completion details: %q", view)
	}
	if strings.Contains(view, "─") || strings.Contains(view, "│") {
		t.Fatalf("editor context should be borderless: %q", view)
	}
}

func TestDatasetContextShowsFullyQualifiedName(t *testing.T) {
	state := initialModelWithProjects(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), []project.Project{{ID: "project-id", Resources: []project.Resource{{Name: "dataset-name", Kind: "dataset"}}}})
	state.width = 80
	state.focus = focusProjects
	state.selectedDataset = 0
	state.selectedChild = -1
	if got := strings.ReplaceAll(strings.ReplaceAll(state.contextInfoView(), " ", ""), "\n", ""); !contains(got, "project-id.dataset-name") {
		t.Fatalf("dataset context should be fully qualified: %q", got)
	}
}

func TestProjectPaneScrollsAtViewportEdges(t *testing.T) {
	children := make([]project.Resource, 20)
	for index := range children {
		children[index] = project.Resource{Name: fmt.Sprintf("table_%02d", index), Kind: "table", DetailsLoaded: true}
	}
	state := initialModelWithProjects(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), []project.Project{{ID: "project-1", Resources: []project.Resource{{Name: "dataset-1", Kind: "dataset", Children: children}}}})
	state.width = 80
	state.height = 20
	state.focus = focusProjects
	state.expanded[0] = true
	state.selectedDataset = 0
	state.expandedDataset[state.datasetKey(0, 0)] = true
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

func TestProjectSelectionContinuesPastExpandedChildren(t *testing.T) {
	state := initialModelWithProjects(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), []project.Project{
		{ID: "project-1", Resources: []project.Resource{
			{Name: "dataset-1", Kind: "dataset", Children: []project.Resource{{Name: "table-1", Kind: "table"}}},
			{Name: "dataset-2", Kind: "dataset", Children: []project.Resource{{Name: "table-2", Kind: "table"}}},
		}},
		{ID: "project-2", Resources: []project.Resource{{Name: "dataset-3", Kind: "dataset"}}},
	})
	state.focus = focusProjects
	state.expanded[0] = true
	state.expandedDataset[state.datasetKey(0, 0)] = true
	state.expandedDataset[state.datasetKey(0, 1)] = true
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
	expected := []string{"QUERY EDITOR", "RESULTS", "RUN HISTORY", "SHORTCUTS", "EXPLORER"}
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

func TestHelpModalShowsCompleteKeymapAndClosesWithoutQuitting(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	state.width, state.height = 100, 24
	state.focus = focusProjects
	updated, command := state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	state = updated.(model)
	if command != nil || !state.showHelp {
		t.Fatal("? should open the help modal without a command")
	}
	view := state.helpView()
	for _, text := range []string{"GENERAL", "EXPLORER"} {
		if !contains(view, text) {
			t.Fatalf("help modal is missing %q: %q", text, view)
		}
	}
	state.helpScroll = 10
	view = state.helpView()
	for _, text := range []string{"ctrl+e", "QUERY EDITOR"} {
		if !contains(view, text) {
			t.Fatalf("scrolled help modal is missing %q: %q", text, view)
		}
	}
	state.helpScroll = 20
	view = state.helpView()
	for _, text := range []string{"COMPLETION", "RESULTS"} {
		if !contains(view, text) {
			t.Fatalf("middle help modal is missing %q: %q", text, view)
		}
	}
	state.helpScroll = 30
	view = state.helpView()
	for _, text := range []string{"RUN HISTORY", "SEARCH"} {
		if !contains(view, text) {
			t.Fatalf("lower help modal is missing %q: %q", text, view)
		}
	}
	state.helpScroll = 100
	view = state.helpView()
	for _, text := range []string{"RESOURCE INFO", "HELP", "Esc/?/q"} {
		if !contains(view, text) {
			t.Fatalf("bottom help modal is missing %q: %q", text, view)
		}
	}
	if lipgloss.Width(view) > state.width || lipgloss.Height(view) > state.height {
		t.Fatalf("help modal exceeded terminal: %dx%d in %dx%d", lipgloss.Width(view), lipgloss.Height(view), state.width, state.height)
	}
	updated, command = state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	state = updated.(model)
	if command != nil || state.showHelp {
		t.Fatal("q should close help without quitting the TUI")
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

func TestProjectPaneUsesQuarterTerminalWidth(t *testing.T) {
	state := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	state.width = 120
	if got, want := state.projectPanelWidth(), 30; got != want {
		t.Fatalf("project pane width = %d, want %d", got, want)
	}
	state.width = 80
	if got, want := state.projectPanelWidth(), 20; got != want {
		t.Fatalf("project pane width = %d, want %d", got, want)
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

func TestResultsPaneFillsAvailableWidth(t *testing.T) {
	state := initialModelWithMock(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}), true)
	updated, _ := state.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	state = updated.(model)
	if got, want := lipgloss.Width(state.resultView()), state.tabs[0].results.Width()+2; got != want {
		t.Fatalf("results pane width = %d, want %d", got, want)
	}
	state.setResult(0, bigquery.Result{Columns: []string{"id", "name"}, Rows: []bigquery.Row{{Values: []string{"1", "Ada"}}}})
	if got, want := lipgloss.Width(state.resultView()), state.tabs[0].results.Width()+2; got != want {
		t.Fatalf("populated results pane width = %d, want %d", got, want)
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
