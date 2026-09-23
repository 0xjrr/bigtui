package app

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xjrr/bigtui/internal/bigquery"
)

func TestResultsKeepHeadersAndRowNumbersWhileNavigating(t *testing.T) {
	state := newMockModel()
	state.focus = focusResults
	state.width, state.height = 100, 30
	state.resizeTab(0)
	state.setResult(0, bigquery.Result{
		Columns: []string{"id", "name", "segment"},
		Rows:    []bigquery.Row{{Values: []string{"1", "Ada", "enterprise"}}, {Values: []string{"2", "Grace", "startup"}}},
	})
	view := state.renderResults()
	if !strings.Contains(view, "id") || !strings.Contains(view, "name") || !strings.Contains(view, "1") {
		t.Fatalf("results should retain headers and row numbers: %q", view)
	}
	state, _ = press(state, tea.KeyDown)
	if state.tabs[0].resultRow != 1 {
		t.Fatalf("down should move result row cursor: %d", state.tabs[0].resultRow)
	}
	state, _ = typeRunes(state, 'l')
	if state.tabs[0].resultColumn != 1 {
		t.Fatalf("l should move result column cursor: %d", state.tabs[0].resultColumn)
	}
}

func TestResultCursorsStayInsideTheResultSet(t *testing.T) {
	state := newMockModel()
	state.focus = focusResults
	state.width, state.height = 100, 30
	state.resizeTab(0)
	state.setResult(0, bigquery.Result{
		Columns: []string{"id", "name"},
		Rows:    []bigquery.Row{{Values: []string{"1", "Ada"}}},
	})
	state.moveResultRow(-5)
	state.moveResultColumn(-5)
	if state.tabs[0].resultRow != 0 || state.tabs[0].resultColumn != 0 {
		t.Fatalf("cursors moved before the first cell: row=%d column=%d", state.tabs[0].resultRow, state.tabs[0].resultColumn)
	}
	state.moveResultRow(5)
	state.moveResultColumn(5)
	if state.tabs[0].resultRow != 0 || state.tabs[0].resultColumn != 1 {
		t.Fatalf("cursors moved past the last cell: row=%d column=%d", state.tabs[0].resultRow, state.tabs[0].resultColumn)
	}
}

func TestResultNavigationIsIgnoredWithoutResults(t *testing.T) {
	state := newMockModel()
	state.focus = focusResults
	state.moveResultRow(1)
	state.moveResultColumn(1)
	if state.tabs[0].resultRow != 0 || state.tabs[0].resultColumn != 0 {
		t.Fatalf("empty results should not move the cursors: row=%d column=%d", state.tabs[0].resultRow, state.tabs[0].resultColumn)
	}
	if state.renderResults() != "Result" {
		t.Fatalf("unexpected placeholder: %q", state.renderResults())
	}
}

func TestResultsScrollHorizontallyToTheSelectedColumn(t *testing.T) {
	state := newMockModel()
	state.width, state.height = 100, 30
	state.resizeTab(0)
	columns := make([]string, 12)
	values := make([]string, 12)
	for index := range columns {
		columns[index] = fmt.Sprintf("column_%02d", index)
		values[index] = fmt.Sprintf("value_%02d", index)
	}
	state.setResult(0, bigquery.Result{Columns: columns, Rows: []bigquery.Row{{Values: values}}})
	for index := 0; index < 11; index++ {
		state.moveResultColumn(1)
	}
	view := state.renderResults()
	if !strings.Contains(view, "column_11") {
		t.Fatalf("the selected column should be visible: %q", view)
	}
	if strings.Contains(view, "column_00") {
		t.Fatalf("earlier columns should scroll out of view: %q", view)
	}
}

func TestResultsViewportDoesNotGrowWithRows(t *testing.T) {
	state := newMockModel()
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

func TestResultsScrollVerticallyToTheSelectedRow(t *testing.T) {
	state := newMockModel()
	state.width, state.height = 100, 30
	state.resizeTab(0)
	rows := make([]bigquery.Row, 100)
	for index := range rows {
		rows[index] = bigquery.Row{Values: []string{fmt.Sprintf("row_%03d", index)}}
	}
	state.setResult(0, bigquery.Result{Columns: []string{"id"}, Rows: rows})
	for index := 0; index < 99; index++ {
		state.moveResultRow(1)
	}
	view := state.renderResults()
	if !strings.Contains(view, "row_099") || strings.Contains(view, "row_000") {
		t.Fatalf("the viewport should follow the selected row: %q", view)
	}
}

func TestResultsPaneFillsAvailableHeight(t *testing.T) {
	state, _ := send(newMockModel(), tea.WindowSizeMsg{Width: 120, Height: 40})
	if got, want := lipgloss.Height(state.resultView()), state.tabs[0].results.Height()+3; got != want {
		t.Fatalf("results pane height = %d, want %d", got, want)
	}
}

func TestResultsPaneFillsAvailableWidth(t *testing.T) {
	state, _ := send(newMockModel(), tea.WindowSizeMsg{Width: 120, Height: 40})
	if got, want := lipgloss.Width(state.resultView()), state.tabs[0].results.Width()+2; got != want {
		t.Fatalf("results pane width = %d, want %d", got, want)
	}
	state.setResult(0, bigquery.Result{Columns: []string{"id", "name"}, Rows: []bigquery.Row{{Values: []string{"1", "Ada"}}}})
	if got, want := lipgloss.Width(state.resultView()), state.tabs[0].results.Width()+2; got != want {
		t.Fatalf("populated results pane width = %d, want %d", got, want)
	}
}
