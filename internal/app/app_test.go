package app

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xjrr/bigtui/internal/bigquery"
)

type clientFunc func(context.Context, string, string) (bigquery.Result, error)

func (f clientFunc) Query(ctx context.Context, projectID, sql string) (bigquery.Result, error) {
	return f(ctx, projectID, sql)
}

func TestInitialQueryUsesRealLineBreaks(t *testing.T) {
	model := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	if value := model.editor.Value(); value == "" || value[0] == '\\' || value[0] == ' ' {
		t.Fatalf("unexpected starter query: %q", value)
	}
	for _, line := range []string{"FROM `demo-analytics", "GROUP BY project_id", "ORDER BY rows DESC"} {
		if !contains(model.editor.Value(), line) {
			t.Fatalf("starter query is missing %q: %q", line, model.editor.Value())
		}
	}
}

func TestQQuitsFromWorkspace(t *testing.T) {
	model := initialModel(clientFunc(func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}))
	_, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if command == nil {
		t.Fatal("expected q to return a quit command")
	}
	if _, ok := command().(tea.QuitMsg); !ok {
		t.Fatalf("expected quit message, got %T", command())
	}
}

func contains(value, part string) bool {
	for index := 0; index+len(part) <= len(value); index++ {
		if value[index:index+len(part)] == part {
			return true
		}
	}
	return false
}
