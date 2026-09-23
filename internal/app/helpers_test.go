package app

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xjrr/bigtui/internal/bigquery"
	"github.com/xjrr/bigtui/internal/project"
)

type clientFunc func(context.Context, string, string) (bigquery.Result, error)

func (f clientFunc) Query(ctx context.Context, projectID, sql string) (bigquery.Result, error) {
	return f(ctx, projectID, sql)
}

type analyzerClient struct {
	clientFunc
	analysis bigquery.Analysis
}

func (a analyzerClient) Analyze(context.Context, string, string) bigquery.Analysis {
	return a.analysis
}

type catalogLoaderStub struct {
	datasetCalls      int
	lastIncludeHidden bool
	tableCalls        int
	resourceCalls     int
	loadErr           error
}

func (s *catalogLoaderStub) Load(context.Context) ([]project.Project, error) {
	return []project.Project{{ID: "project-1", Name: "Project 1"}}, s.loadErr
}

func (s *catalogLoaderStub) LoadDatasets(_ context.Context, _ string, includeHidden bool) ([]project.Resource, error) {
	s.datasetCalls++
	s.lastIncludeHidden = includeHidden
	return []project.Resource{{Name: "dataset-1", Kind: "dataset"}}, nil
}

func (s *catalogLoaderStub) LoadTables(context.Context, string, string) ([]project.Resource, error) {
	s.tableCalls++
	return []project.Resource{{Name: "table-1", Kind: "table"}}, nil
}

func (s *catalogLoaderStub) LoadResource(_ context.Context, _, _, tableID string) (project.Resource, error) {
	s.resourceCalls++
	return project.Resource{Name: tableID, Kind: "table", DetailsLoaded: true}, nil
}

func stubClient() clientFunc {
	return func(context.Context, string, string) (bigquery.Result, error) {
		return bigquery.Result{}, nil
	}
}

func newModel() model {
	return initialModel(stubClient())
}

func newMockModel() model {
	return initialModelWithMock(stubClient(), true)
}

func newModelWithProjects(projects []project.Project) model {
	return initialModelWithProjects(stubClient(), projects)
}

func newModelWithLoader(projects []project.Project, loader project.CatalogLoader) model {
	return initialModelWithProjectsAndLoader(stubClient(), projects, loader)
}

func send(state model, msg tea.Msg) (model, tea.Cmd) {
	updated, command := state.Update(msg)
	return updated.(model), command
}

func press(state model, keyType tea.KeyType) (model, tea.Cmd) {
	return send(state, tea.KeyMsg{Type: keyType})
}

func typeRunes(state model, runes ...rune) (model, tea.Cmd) {
	return send(state, tea.KeyMsg{Type: tea.KeyRunes, Runes: runes})
}

func runCommand(t *testing.T, state model, command tea.Cmd) model {
	t.Helper()
	if command == nil {
		t.Fatal("expected a command to run")
	}
	updated, _ := send(state, command())
	return updated
}

func isQuitCommand(command tea.Cmd) bool {
	if command == nil {
		return false
	}
	_, ok := command().(tea.QuitMsg)
	return ok
}
