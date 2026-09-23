package theme

import "github.com/charmbracelet/lipgloss"

const (
	Accent = lipgloss.Color("#f4b860")
	Ink    = lipgloss.Color("#eef0f2")
	Muted  = lipgloss.Color("#89919a")
	Panel  = lipgloss.Color("#20262b")
	Border = lipgloss.Color("#3a444c")
)

func Title(value string) string {
	return lipgloss.NewStyle().Foreground(Accent).Bold(true).Render(value)
}

func Selected(value string) string {
	return lipgloss.NewStyle().Foreground(Accent).Bold(true).Render(value)
}

func Emphasis(value string) string {
	return lipgloss.NewStyle().Foreground(Accent).Render(value)
}

func Dim(value string) string {
	return lipgloss.NewStyle().Foreground(Muted).Render(value)
}

func Bright(value string) string {
	return lipgloss.NewStyle().Foreground(Ink).Render(value)
}

func Strong(value string) string {
	return lipgloss.NewStyle().Foreground(Ink).Bold(true).Render(value)
}

func Box(focused bool) lipgloss.Style {
	color := Border
	if focused {
		color = Accent
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(color)
}

func Modal(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Accent).
		Padding(2)
}
