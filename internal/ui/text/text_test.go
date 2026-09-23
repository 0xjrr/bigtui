package text

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTruncateKeepsShortValues(t *testing.T) {
	cases := map[string]struct {
		value string
		width int
		want  string
	}{
		"fits":        {value: "orders", width: 10, want: "orders"},
		"exact":       {value: "orders", width: 6, want: "orders"},
		"too narrow":  {value: "orders", width: 3, want: "orders"},
		"long value":  {value: "project-33b62768-e4b9-483c-a1b", width: 18, want: "project-33b6276..."},
		"minimum cut": {value: "orders", width: 4, want: "o..."},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			got := Truncate(testCase.value, testCase.width)
			if got != testCase.want {
				t.Fatalf("Truncate(%q, %d) = %q, want %q", testCase.value, testCase.width, got, testCase.want)
			}
			if strings.Contains(got, "\n") {
				t.Fatalf("truncated value contains a newline: %q", got)
			}
		})
	}
}

func TestWrapSplitsLongLines(t *testing.T) {
	lines := Wrap("SELECT customer_id, customer_name FROM customer_order_totals", 20)
	if len(lines) < 2 {
		t.Fatalf("expected long text to wrap: %#v", lines)
	}
	for _, line := range lines {
		if lipgloss.Width(line) > 20 {
			t.Fatalf("wrapped line exceeds width: %q", line)
		}
	}
}

func TestWrapKeepsExistingLineBreaks(t *testing.T) {
	lines := Wrap("SELECT 1\nFROM t", 40)
	if len(lines) != 2 || lines[0] != "SELECT 1" || lines[1] != "FROM t" {
		t.Fatalf("unexpected wrap result: %#v", lines)
	}
}

func TestWrapReturnsTheValueForInvalidWidths(t *testing.T) {
	lines := Wrap("SELECT 1", 0)
	if len(lines) != 1 || lines[0] != "SELECT 1" {
		t.Fatalf("unexpected wrap result: %#v", lines)
	}
}

func TestWrapLinesPreservesBlankLines(t *testing.T) {
	lines := WrapLines([]string{"one", "", "a much longer line of text"}, 10)
	if lines[0] != "one" || lines[1] != "" {
		t.Fatalf("unexpected wrapped lines: %#v", lines)
	}
	if len(lines) < 4 {
		t.Fatalf("long lines should wrap onto several rows: %#v", lines)
	}
}

func TestOverlaySplicesPopupAtGivenPosition(t *testing.T) {
	base := "AAAAAAAAAA\nBBBBBBBBBB\nCCCCCCCCCC"
	got := Overlay(base, "XY\nZW", 1, 3)
	want := "AAAAAAAAAA\nBBBXYBBBBB\nCCCZWCCCCC"
	if got != want {
		t.Fatalf("overlay placed popup incorrectly:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestOverlayPadsShortLinesAndIgnoresOutOfRangeRows(t *testing.T) {
	got := Overlay("A\nB", "XYZ", 0, 3)
	if got != "A  XYZ\nB" {
		t.Fatalf("overlay should pad short lines before the popup column: %q", got)
	}
	got = Overlay("A", "XYZ", 5, 0)
	if got != "A" {
		t.Fatalf("overlay should ignore rows outside the base content: %q", got)
	}
	got = Overlay("A", "XYZ", -3, 0)
	if got != "A" {
		t.Fatalf("overlay should ignore negative rows: %q", got)
	}
}

func TestOverlayKeepsTheBaseHeight(t *testing.T) {
	base := strings.Repeat("ABCDE\n", 5)
	if got := Overlay(base, "X\nY", 2, 1); lipgloss.Height(got) != lipgloss.Height(base) {
		t.Fatalf("overlay changed the height from %d to %d", lipgloss.Height(base), lipgloss.Height(got))
	}
}
