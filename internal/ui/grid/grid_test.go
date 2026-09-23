package grid

import (
	"strings"
	"testing"
)

func TestRowPadsAndTruncatesValues(t *testing.T) {
	got := Row([]string{"ab", "a-very-long-value"}, []int{5, 8})
	if got != "ab     a-ver..." {
		t.Fatalf("unexpected row: %q", got)
	}
}

func TestRowFillsMissingValues(t *testing.T) {
	if got := Row([]string{"ab"}, []int{4, 4}); got != "ab" {
		t.Fatalf("missing values should render as trailing blanks: %q", got)
	}
}

func TestSeparatorMatchesColumnWidths(t *testing.T) {
	if got := Separator([]int{3, 2}); got != "---  --" {
		t.Fatalf("unexpected separator: %q", got)
	}
}

func TestWindowExtractsColumnRange(t *testing.T) {
	values := []string{"a", "b", "c"}
	if got := Window(values, 1, 3); len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("unexpected window: %#v", got)
	}
	if got := Window(values, 2, 5); len(got) != 3 || got[0] != "c" || got[1] != "" {
		t.Fatalf("window beyond the row should pad with blanks: %#v", got)
	}
}

func TestPreviewIsStructured(t *testing.T) {
	preview := Preview([]string{"id", "name"}, [][]string{{"1", "Ada"}})
	if len(preview) != 3 {
		t.Fatalf("unexpected preview shape: %#v", preview)
	}
	if !strings.Contains(preview[0], "id") || !strings.Contains(preview[1], "-") || !strings.Contains(preview[2], "Ada") {
		t.Fatalf("unexpected structured preview: %#v", preview)
	}
}

func TestPreviewWithoutColumnsIsEmpty(t *testing.T) {
	if preview := Preview(nil, [][]string{{"1"}}); preview != nil {
		t.Fatalf("expected no preview without columns: %#v", preview)
	}
}

func TestPreviewColumnsGrowWithContentUpToTheLimit(t *testing.T) {
	preview := Preview([]string{"id"}, [][]string{{strings.Repeat("x", 40)}})
	separator := strings.TrimRight(strings.TrimPrefix(preview[1], "  "), " ")
	if len(separator) != previewColumnWidth {
		t.Fatalf("preview column width = %d, want %d", len(separator), previewColumnWidth)
	}
}
