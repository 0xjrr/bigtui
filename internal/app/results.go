package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xjrr/bigtui/internal/bigquery"
	"github.com/0xjrr/bigtui/internal/ui/grid"
	"github.com/0xjrr/bigtui/internal/ui/text"
	"github.com/0xjrr/bigtui/internal/ui/theme"
)

const (
	rowNumberWidth       = 5
	resultColumnMaxWidth = 20
	resultColumnMinWidth = 8
)

func (m *model) setResult(tabIndex int, result bigquery.Result) {
	m.tabs[tabIndex].result = result
	width := max(6, (m.tabs[tabIndex].results.Width()-2*max(0, len(result.Columns)-1))/max(1, len(result.Columns)))
	columns := make([]table.Column, len(result.Columns))
	for index, column := range result.Columns {
		columns[index] = table.Column{Title: text.Truncate(column, width), Width: width}
	}
	rows := make([]table.Row, len(result.Rows))
	for index, row := range result.Rows {
		rows[index] = table.Row(row.Values)
	}
	m.tabs[tabIndex].results.SetColumns(columns)
	m.tabs[tabIndex].results.SetRows(rows)
	m.resizeResultColumns(tabIndex)
}

func (m *model) resizeResultColumns(tabIndex int) {
	columns := m.tabs[tabIndex].results.Columns()
	if len(columns) == 0 {
		return
	}
	width := max(6, (m.tabs[tabIndex].results.Width()-2*max(0, len(columns)-1))/len(columns))
	for index := range columns {
		columns[index].Width = width
	}
	m.tabs[tabIndex].results.SetColumns(columns)
}

func (m *model) moveResultRow(direction int) {
	tab := m.activeQueryTab()
	if len(tab.result.Rows) == 0 {
		return
	}
	tab.resultRow = clamp(tab.resultRow+direction, 0, len(tab.result.Rows)-1)
}

func (m *model) moveResultColumn(direction int) {
	tab := m.activeQueryTab()
	if len(tab.result.Columns) == 0 {
		return
	}
	tab.resultColumn = clamp(tab.resultColumn+direction, 0, len(tab.result.Columns)-1)
	tab.resultOffset = tab.resultColumn
}

func (m model) resultView() string {
	height := max(3, m.tabs[m.activeTab].results.Height())
	width := max(10, m.tabs[m.activeTab].results.Width())
	box := theme.Box(m.focus == focusResults).Width(width).Height(height).Render(m.renderResults())
	return lipgloss.JoinVertical(lipgloss.Left, theme.Title("RESULTS"), box)
}

func (m model) renderResults() string {
	tab := m.tabs[m.activeTab]
	if len(tab.result.Columns) == 0 {
		return "Result"
	}
	widths := resultColumnWidths(tab.result)
	offset, end := visibleResultColumns(tab, widths)
	padding := strings.Repeat(" ", rowNumberWidth)
	lines := []string{
		padding + grid.Row(tab.result.Columns[offset:end], widths[offset:end]),
		padding + grid.Separator(widths[offset:end]),
	}
	start, stop := visibleResultRows(tab)
	for index := start; index < stop; index++ {
		values := grid.Window(tab.result.Rows[index].Values, offset, end)
		line := fmt.Sprintf("%4d ", index+1) + grid.Row(values, widths[offset:end])
		if tab.resultRow == index {
			line = theme.Emphasis(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func resultColumnWidths(result bigquery.Result) []int {
	widths := make([]int, len(result.Columns))
	for index, column := range result.Columns {
		widths[index] = max(resultColumnMinWidth, min(resultColumnMaxWidth, len(column)))
		for _, row := range result.Rows {
			if index >= len(row.Values) {
				continue
			}
			widths[index] = max(widths[index], min(resultColumnMaxWidth, len(row.Values[index])))
		}
	}
	return widths
}

func visibleResultColumns(tab queryTab, widths []int) (int, int) {
	offset := max(tab.resultOffset, tab.resultColumn)
	available := tab.results.Width() - rowNumberWidth
	used := 0
	end := offset
	for end < len(widths) && used+widths[end]+2 <= available {
		used += widths[end] + 2
		end++
	}
	if end == offset {
		end++
	}
	return offset, end
}

func visibleResultRows(tab queryTab) (int, int) {
	viewport := max(1, tab.results.Height()-2)
	start := 0
	if tab.resultRow >= viewport {
		start = tab.resultRow - viewport + 1
	}
	return start, min(len(tab.result.Rows), start+viewport)
}

func clamp(value, low, high int) int {
	return min(max(value, low), high)
}
