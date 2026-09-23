package theme

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestHelpersKeepTheRenderedText(t *testing.T) {
	renderers := map[string]func(string) string{
		"title":    Title,
		"selected": Selected,
		"emphasis": Emphasis,
		"dim":      Dim,
		"bright":   Bright,
		"strong":   Strong,
	}
	for name, render := range renderers {
		if got := render("EXPLORER"); !strings.Contains(got, "EXPLORER") {
			t.Fatalf("%s renderer dropped its content: %q", name, got)
		}
	}
}

func TestBoxDrawsARoundedBorder(t *testing.T) {
	rendered := Box(true).Render("content")
	if !strings.Contains(rendered, "╭") || !strings.Contains(rendered, "╯") {
		t.Fatalf("box should draw a rounded border: %q", rendered)
	}
	if lipgloss.Height(rendered) != 3 {
		t.Fatalf("box height = %d, want 3", lipgloss.Height(rendered))
	}
}

func TestModalUsesTheRequestedSize(t *testing.T) {
	rendered := Modal(40, 10).Render("content")
	if lipgloss.Width(rendered) != 42 || lipgloss.Height(rendered) != 12 {
		t.Fatalf("modal size = %dx%d, want 42x12 including borders", lipgloss.Width(rendered), lipgloss.Height(rendered))
	}
}
