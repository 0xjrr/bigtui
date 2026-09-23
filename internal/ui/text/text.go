package text

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func Truncate(value string, width int) string {
	if width < 4 || len(value) <= width {
		return value
	}
	return value[:width-3] + "..."
}

func Wrap(value string, width int) []string {
	if width < 1 {
		return []string{value}
	}
	var lines []string
	for _, sourceLine := range strings.Split(value, "\n") {
		lines = append(lines, wrapLine(sourceLine, width)...)
	}
	return lines
}

func WrapLines(lines []string, width int) []string {
	wrapped := []string{}
	for _, line := range lines {
		value := ansi.Wrap(line, width, "")
		if value == "" {
			wrapped = append(wrapped, "")
			continue
		}
		wrapped = append(wrapped, strings.Split(value, "\n")...)
	}
	return wrapped
}

func Overlay(base, popup string, row, col int) string {
	baseLines := strings.Split(base, "\n")
	popupLines := strings.Split(popup, "\n")
	popupWidth := lipgloss.Width(popup)
	for index, popupLine := range popupLines {
		target := row + index
		if target < 0 || target >= len(baseLines) {
			continue
		}
		baseLines[target] = spliceLine(baseLines[target], popupLine, col, popupWidth)
	}
	return strings.Join(baseLines, "\n")
}

func wrapLine(value string, width int) []string {
	lines := []string{}
	line := ""
	for _, character := range value {
		candidate := line + string(character)
		if line != "" && lipgloss.Width(candidate) > width {
			lines = append(lines, line)
			line = string(character)
			continue
		}
		line = candidate
	}
	return append(lines, line)
}

func spliceLine(line, popupLine string, col, popupWidth int) string {
	lineWidth := lipgloss.Width(line)
	left := ansi.Cut(line, 0, min(col, lineWidth))
	if pad := col - lipgloss.Width(left); pad > 0 {
		left += strings.Repeat(" ", pad)
	}
	right := ""
	if col+popupWidth < lineWidth {
		right = ansi.Cut(line, col+popupWidth, lineWidth)
	}
	return left + popupLine + right
}
