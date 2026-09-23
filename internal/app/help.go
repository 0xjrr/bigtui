package app

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/0xjrr/bigtui/internal/ui/text"
	"github.com/0xjrr/bigtui/internal/ui/theme"
)

func (m model) helpView() string {
	modalWidth := max(50, min(90, m.width-4))
	modalHeight := max(12, m.height-4)
	lines := text.WrapLines(helpLines(), max(20, modalWidth-6))
	visible := scrollWindow(lines, m.helpScroll, max(1, max(1, modalHeight-6)-1))
	visible = append(visible, "", theme.Bright("Up/Down scroll  ·  Esc/?/q close"))
	return theme.Modal(max(1, modalWidth-2), max(1, modalHeight-2)).Render(lipgloss.JoinVertical(lipgloss.Left, visible...))
}

func helpLines() []string {
	return []string{
		theme.Title("GENERAL"),
		"tab / shift+tab     move focus",
		"ctrl+left/right     switch query tab",
		"ctrl+shift+tab     switch query tab",
		"ctrl+n             new query tab",
		"ctrl+w             close query tab",
		"ctrl+s             search resources",
		"ctrl+c             quit",
		"q                  quit outside editor",
		"?                  open/close help",
		"",
		theme.Title("EXPLORER"),
		"up/down            select project, dataset, table, or view",
		"left/right         collapse or expand",
		"ctrl+b             set billing project",
		"ctrl+e             insert SELECT query for selected resource",
		"ctrl+h             show/hide hidden datasets",
		"enter              inspect selected resource",
		"",
		theme.Title("QUERY EDITOR"),
		"typing             update autocomplete",
		"ctrl+r             run query",
		"ctrl+enter         run query",
		"ctrl+j             run query",
		"enter              insert newline",
		"",
		theme.Title("COMPLETION"),
		"up/down            select suggestion",
		"ctrl+k/ctrl+j      select suggestion",
		"enter              accept suggestion",
		"esc                close suggestions",
		"left/right         close suggestions",
		"",
		theme.Title("RESULTS"),
		"up/down            move through rows",
		"left/right         move through columns",
		"",
		theme.Title("RUN HISTORY"),
		"up/down            select run",
		"enter              load selected query",
		"",
		theme.Title("SEARCH"),
		"ctrl+s             open/close search",
		"up/down            select result",
		"ctrl+k/ctrl+j      select result",
		"enter              select result",
		"esc                close search",
		"",
		theme.Title("RESOURCE INFO"),
		"up/down            scroll one line",
		"pgup/pgdown        scroll one page",
		"enter/esc/q        close info",
		"",
		theme.Title("HELP"),
		"up/down            scroll help",
		"pgup/pgdown        scroll one page",
		"esc/?/q            close help and return to the TUI",
	}
}
