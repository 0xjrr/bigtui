package grid

import (
	"fmt"
	"strings"

	"github.com/xjrr/bigtui/internal/ui/text"
)

const previewColumnWidth = 18

func Row(values []string, widths []int) string {
	parts := make([]string, len(widths))
	for index, width := range widths {
		value := ""
		if index < len(values) {
			value = text.Truncate(values[index], width)
		}
		parts[index] = fmt.Sprintf("%-*s", width, value)
	}
	return strings.TrimRight(strings.Join(parts, "  "), " ")
}

func Separator(widths []int) string {
	parts := make([]string, len(widths))
	for index, width := range widths {
		parts[index] = strings.Repeat("-", width)
	}
	return strings.Join(parts, "  ")
}

func Window(values []string, start, end int) []string {
	window := make([]string, end-start)
	for index := range window {
		if start+index < len(values) {
			window[index] = values[start+index]
		}
	}
	return window
}

func Preview(columns []string, rows [][]string) []string {
	if len(columns) == 0 {
		return nil
	}
	widths := previewWidths(columns, rows)
	lines := []string{"  " + Row(columns, widths), "  " + Separator(widths)}
	for _, row := range rows {
		lines = append(lines, "  "+Row(row, widths))
	}
	return lines
}

func previewWidths(columns []string, rows [][]string) []int {
	widths := make([]int, len(columns))
	for index, column := range columns {
		widths[index] = min(previewColumnWidth, max(4, len(column)))
	}
	for _, row := range rows {
		for index, value := range row {
			if index >= len(widths) {
				continue
			}
			widths[index] = min(previewColumnWidth, max(widths[index], len(value)))
		}
	}
	return widths
}
