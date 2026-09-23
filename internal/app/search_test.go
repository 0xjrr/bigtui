package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCtrlSOpensQualifiedResourceSearch(t *testing.T) {
	state := newMockModel()
	state, _ = press(state, tea.KeyCtrlS)
	if !state.searchOpen || len(state.searchResults) == 0 {
		t.Fatal("ctrl+s should open a search with resources")
	}
	state, _ = typeRunes(state, 'c')
	if !strings.Contains(state.searchView(), "customers") || !strings.Contains(state.searchView(), "sandbox-analytics") {
		t.Fatalf("search should show qualified resource locations: %q", state.searchView())
	}
	state, _ = press(state, tea.KeyEnter)
	if state.searchOpen || state.focus != focusProjects {
		t.Fatal("enter should select a search result and return to Projects focus")
	}
}

func TestSearchFiltersAcrossCatalogLevels(t *testing.T) {
	state := newMockModel()
	state, _ = press(state, tea.KeyCtrlS)
	all := len(state.searchResults)
	state, _ = typeRunes(state, 'f')
	state, _ = typeRunes(state, 'i')
	if len(state.searchResults) == 0 || len(state.searchResults) >= all {
		t.Fatalf("typing should narrow the result list: %d of %d", len(state.searchResults), all)
	}
	for _, result := range state.searchResults {
		joined := strings.ToLower(result.name + result.project + result.projectID + result.dataset)
		if !strings.Contains(joined, "fi") {
			t.Fatalf("unexpected search result: %#v", result)
		}
	}
}

func TestSearchReportsWhenNothingMatches(t *testing.T) {
	state := newMockModel()
	state.width, state.height = 100, 30
	state, _ = press(state, tea.KeyCtrlS)
	state.searchInput.SetValue("zzzz-no-match")
	state.refreshSearch()
	if len(state.searchResults) != 0 {
		t.Fatalf("expected no matches: %#v", state.searchResults)
	}
	if !strings.Contains(state.searchView(), "No matching resources.") {
		t.Fatalf("search should report an empty result set: %q", state.searchView())
	}
}

func TestSearchCursorStaysInsideTheResultList(t *testing.T) {
	state := newMockModel()
	state, _ = press(state, tea.KeyCtrlS)
	state, _ = press(state, tea.KeyUp)
	if state.searchCursor != 0 {
		t.Fatalf("up at the first result = %d, want 0", state.searchCursor)
	}
	for index := 0; index < len(state.searchResults)+3; index++ {
		state, _ = press(state, tea.KeyDown)
	}
	if state.searchCursor != len(state.searchResults)-1 {
		t.Fatalf("down past the last result = %d, want %d", state.searchCursor, len(state.searchResults)-1)
	}
}

func TestSelectingATableExpandsItsPathInTheExplorer(t *testing.T) {
	state := newMockModel()
	state, _ = press(state, tea.KeyCtrlS)
	state.searchInput.SetValue("customers")
	state.refreshSearch()
	for index, result := range state.searchResults {
		if result.childIndex >= 0 {
			state.searchCursor = index
			break
		}
	}
	selected := state.searchResults[state.searchCursor]
	state.selectSearchResult()
	if !state.expanded[selected.projectIndex] || !state.datasetExpanded(selected.projectIndex, selected.datasetIndex) {
		t.Fatal("selecting a table should expand its project and dataset")
	}
	if state.selectedChild != selected.childIndex || state.status != "Selected "+selected.name {
		t.Fatalf("unexpected selection state: child=%d status=%q", state.selectedChild, state.status)
	}
}

func TestEscapeClosesSearchWithoutSelecting(t *testing.T) {
	state := newMockModel()
	state, _ = press(state, tea.KeyCtrlS)
	active := state.active
	state, _ = press(state, tea.KeyEscape)
	if state.searchOpen || state.active != active {
		t.Fatalf("esc should close search without changing the selection: open=%v active=%d", state.searchOpen, state.active)
	}
}
