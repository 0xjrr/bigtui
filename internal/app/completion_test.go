package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xjrr/bigtui/internal/completion"
	"github.com/0xjrr/bigtui/internal/project"
)

func TestAutocompleteLoadsDatasetsOnlyForResourceContext(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := newModelWithLoader([]project.Project{{ID: "project-1"}}, loader)
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
	state = runCommand(t, state, command)
	if loader.datasetCalls != 1 || loader.lastIncludeHidden {
		t.Fatalf("unexpected default dataset request: calls=%d includeHidden=%v", loader.datasetCalls, loader.lastIncludeHidden)
	}
}

func TestAutocompleteLoadsTablesForQualifiedDataset(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := newModelWithLoader([]project.Project{{ID: "project-1", Resources: []project.Resource{{Name: "dataset-1", Kind: "dataset"}}}}, loader)
	state.focus = focusEditor
	state.datasetsLoaded[0] = true
	state.tabs[0].editor.SetValue("SELECT * FROM `project-1`.`dataset-1`.")
	state.tabs[0].editor.CursorEnd()
	command := state.refreshCompletion()
	if command == nil {
		t.Fatal("qualified dataset completion should request tables on demand")
	}
	state = runCommand(t, state, command)
	if loader.tableCalls != 1 {
		t.Fatalf("expected one table request, got %d", loader.tableCalls)
	}
	if !state.completionOpen || len(state.completionItems) == 0 || state.completionItems[0].Label != "table-1" {
		t.Fatalf("expected table completion after loading: %#v", state.completionItems)
	}
}

func TestAutocompleteDatasetLoadingHonorsHiddenDatasetToggle(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := newModelWithLoader([]project.Project{{ID: "project-1"}}, loader)
	state.focus = focusEditor
	state.showHiddenDatasets = true
	state.tabs[0].editor.SetValue("SELECT * FROM d")
	state.tabs[0].editor.CursorEnd()
	command := state.refreshCompletion()
	if command == nil {
		t.Fatal("resource completion should request datasets")
	}
	state = runCommand(t, state, command)
	if !loader.lastIncludeHidden {
		t.Fatal("autocomplete should include hidden datasets when Ctrl+H mode is active")
	}
}

func TestAutocompleteDoesNotRepeatInFlightCatalogRequests(t *testing.T) {
	loader := &catalogLoaderStub{}
	state := newModelWithLoader([]project.Project{{ID: "project-1"}}, loader)
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("SELECT * FROM d")
	state.tabs[0].editor.CursorEnd()
	if command := state.refreshCompletion(); command == nil {
		t.Fatal("the first resource completion should request datasets")
	}
	if command := state.refreshCompletion(); command != nil {
		t.Fatal("a pending dataset request should not be repeated")
	}
}

func TestTypingOpensCompletionAndEnterInserts(t *testing.T) {
	state := newMockModel()
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("SELECT * FROM cus")
	state.tabs[0].editor.CursorEnd()
	if state.completionOpen {
		t.Fatal("completion should not be open before typing")
	}
	state, _ = typeRunes(state, 't')
	if !state.completionOpen || len(state.completionItems) == 0 {
		t.Fatalf("typing a matching prefix should open completion automatically: open=%v items=%d", state.completionOpen, len(state.completionItems))
	}
	for state.completionItems[state.completionCursor].Label != "customers" {
		if state.completionCursor >= len(state.completionItems)-1 {
			break
		}
		state, _ = press(state, tea.KeyDown)
	}
	found := false
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
	state, _ = press(state, tea.KeyEnter)
	if state.completionOpen {
		t.Fatal("enter should close the completion popup")
	}
	if !strings.HasSuffix(state.tabs[0].editor.Value(), "customers") {
		t.Fatalf("enter should insert the selected completion in place of the typed prefix: %q", state.tabs[0].editor.Value())
	}
}

func TestTabAcceptsHighlightedCompletionWithoutChangingFocus(t *testing.T) {
	state := newMockModel()
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("SELECT * FROM cus")
	state.tabs[0].editor.CursorEnd()
	state.completionItems = []completion.Item{{Label: "customers", InsertText: "customers"}}
	state.completionCursor = 0
	state.completionOpen = true

	state, _ = press(state, tea.KeyTab)

	if state.focus != focusEditor {
		t.Fatalf("tab changed focus to %v, want editor", state.focus)
	}
	if state.completionOpen {
		t.Fatal("tab should close the completion popup")
	}
	if got := state.tabs[0].editor.Value(); got != "SELECT * FROM customers" {
		t.Fatalf("tab accepted completion as %q, want %q", got, "SELECT * FROM customers")
	}
}

func TestSFAcceptanceInsertsSelectFromSnippet(t *testing.T) {
	state := newModel()
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("s")
	state.tabs[0].editor.CursorEnd()
	state, _ = typeRunes(state, 'f')
	if !state.completionOpen || len(state.completionItems) == 0 || state.completionItems[0].InsertText != "SELECT * FROM `" {
		t.Fatalf("expected sf snippet suggestion: %#v", state.completionItems)
	}
	state, _ = press(state, tea.KeyEnter)
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
		{name: "plain keyword", value: "SEL", item: completion.Item{Label: "SELECT", Detail: "keyword", InsertText: "SELECT"}, want: "SELECT"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := newModel()
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
	state := newMockModel()
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

func TestAcceptingWithoutSuggestionsKeepsTheQuery(t *testing.T) {
	state := newModel()
	state.tabs[0].editor.SetValue("SELECT 1")
	state.tabs[0].editor.CursorEnd()
	if command := state.acceptCompletion(); command != nil {
		t.Fatal("accepting an empty completion list should not schedule work")
	}
	if got := state.tabs[0].editor.Value(); got != "SELECT 1" {
		t.Fatalf("query changed to %q", got)
	}
}

func TestCompletionClosesWhenPrefixBecomesEmpty(t *testing.T) {
	state := newMockModel()
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("SELECT * FROM cust")
	state.tabs[0].editor.CursorEnd()
	state, _ = typeRunes(state, 'o')
	if !state.completionOpen {
		t.Fatal("expected completion open after typing a matching prefix")
	}
	state, _ = send(state, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	if state.completionOpen {
		t.Fatal("completion should close once the prefix becomes empty")
	}
}

func TestEscClosesCompletionPopupWithoutChangingQuery(t *testing.T) {
	state := newMockModel()
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("SELECT * FROM cust")
	state.tabs[0].editor.CursorEnd()
	state, _ = typeRunes(state, 'o')
	if !state.completionOpen {
		t.Fatal("expected completion popup to open")
	}
	value := state.tabs[0].editor.Value()
	state, _ = press(state, tea.KeyEscape)
	if state.completionOpen || len(state.completionItems) != 0 {
		t.Fatal("esc should close the completion popup")
	}
	if state.tabs[0].editor.Value() != value {
		t.Fatalf("esc should not modify the query: got %q, want %q", state.tabs[0].editor.Value(), value)
	}
}

func TestArrowKeysCloseTheCompletionPopup(t *testing.T) {
	state := newMockModel()
	state.focus = focusEditor
	state.completionOpen = true
	state.completionItems = []completion.Item{{Label: "customers"}}
	command, stop := state.handleCompletionKey("left")
	if command != nil || stop {
		t.Fatal("left should stay available to the editor")
	}
	if state.completionOpen || len(state.completionItems) != 0 {
		t.Fatal("left should close the completion popup")
	}
}

func TestCompletionCursorStaysInsideTheSuggestionList(t *testing.T) {
	state := newMockModel()
	state.focus = focusEditor
	state.completionOpen = true
	state.completionItems = []completion.Item{{Label: "a"}, {Label: "b"}}
	state, _ = press(state, tea.KeyUp)
	if state.completionCursor != 0 {
		t.Fatalf("up at the first suggestion = %d, want 0", state.completionCursor)
	}
	state, _ = press(state, tea.KeyDown)
	state, _ = press(state, tea.KeyDown)
	if state.completionCursor != 1 {
		t.Fatalf("down past the last suggestion = %d, want 1", state.completionCursor)
	}
}

func TestCompletionSuggestionsAreCapped(t *testing.T) {
	state := newMockModel()
	items := state.completionItemsFor(completion.Request{Project: "sandbox-analytics", SQL: "SELECT c", Cursor: 8})
	if len(items) > completionLimit {
		t.Fatalf("completion returned %d items, want at most %d", len(items), completionLimit)
	}
}

func TestCursorOffsetCountsPreviousLines(t *testing.T) {
	state := newModel()
	state.tabs[0].editor.SetValue("SELECT 1\nFROM t")
	state.tabs[0].editor.CursorEnd()
	if got, want := cursorOffset(state.tabs[0].editor), len("SELECT 1\nFROM t"); got != want {
		t.Fatalf("cursorOffset = %d, want %d", got, want)
	}
}

func TestOverlayCompletionKeepsTheViewHeight(t *testing.T) {
	state := newMockModel()
	state.width, state.height = 120, 40
	state.focus = focusEditor
	state.tabs[0].editor.SetValue("SELECT * FROM cust")
	state.tabs[0].editor.CursorEnd()
	baseView := state.View()
	state, _ = typeRunes(state, 'o')
	if !state.completionOpen {
		t.Fatal("expected completion popup to open")
	}
	if strings.Contains(state.editorView(), "customers") {
		t.Fatal("completion suggestions should not be appended inline in the editor panel")
	}
	overlaid := state.View()
	if !strings.Contains(overlaid, "customers") {
		t.Fatalf("overlaid view should contain the completion popup: %q", overlaid)
	}
	if lipgloss.Height(overlaid) != lipgloss.Height(baseView) {
		t.Fatalf("overlay should not change the rendered view height: got %d, want %d", lipgloss.Height(overlaid), lipgloss.Height(baseView))
	}
}
