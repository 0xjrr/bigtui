package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/0xjrr/bigtui/internal/bigquery"
	"github.com/0xjrr/bigtui/internal/project"
)

type queryFinished struct {
	result  bigquery.Result
	err     error
	tab     int
	history int
}

type queryAnalyzed struct {
	analysis bigquery.Analysis
}

type projectsLoaded struct {
	projects []project.Project
	err      error
}

type datasetsLoaded struct {
	projectIndex  int
	includeHidden bool
	resources     []project.Resource
	err           error
}

type tablesLoaded struct {
	projectIndex int
	datasetIndex int
	resources    []project.Resource
	err          error
}

type resourceLoaded struct {
	projectIndex int
	datasetIndex int
	childIndex   int
	resource     project.Resource
	err          error
}

func (m model) runQuery(historyIndex int) tea.Cmd {
	client := m.client
	projectID := m.billingProjectID()
	tabIndex := m.activeTab
	sql := m.tabs[tabIndex].editor.Value()
	return func() tea.Msg {
		result, err := client.Query(context.Background(), projectID, sql)
		return queryFinished{result: result, err: err, tab: tabIndex, history: historyIndex}
	}
}

func (m model) analyzeQuery() tea.Cmd {
	analyzer, ok := m.client.(bigquery.Analyzer)
	if !ok || len(m.projects) == 0 {
		return nil
	}
	projectID := m.billingProjectID()
	sql := m.tabs[m.activeTab].editor.Value()
	return func() tea.Msg {
		return queryAnalyzed{analysis: analyzer.Analyze(context.Background(), projectID, sql)}
	}
}

func (m model) loadProjects() tea.Cmd {
	loader := m.loader
	return func() tea.Msg {
		projects, err := loader.Load(context.Background())
		return projectsLoaded{projects: projects, err: err}
	}
}

func (m model) loadDatasets(projectIndex int) tea.Cmd {
	loader := m.loader
	projectID := m.projects[projectIndex].ID
	includeHidden := m.showHiddenDatasets
	return func() tea.Msg {
		resources, err := loader.LoadDatasets(context.Background(), projectID, includeHidden)
		return datasetsLoaded{projectIndex: projectIndex, includeHidden: includeHidden, resources: resources, err: err}
	}
}

func (m model) loadTables(projectIndex, datasetIndex int) tea.Cmd {
	loader := m.loader
	projectID := m.projects[projectIndex].ID
	datasetID := m.projects[projectIndex].Resources[datasetIndex].Name
	return func() tea.Msg {
		resources, err := loader.LoadTables(context.Background(), projectID, datasetID)
		return tablesLoaded{projectIndex: projectIndex, datasetIndex: datasetIndex, resources: resources, err: err}
	}
}

func (m model) loadResource(projectIndex, datasetIndex, childIndex int) tea.Cmd {
	loader := m.loader
	projectID := m.projects[projectIndex].ID
	dataset := m.projects[projectIndex].Resources[datasetIndex]
	tableID := dataset.Children[childIndex].Name
	return func() tea.Msg {
		resource, err := loader.LoadResource(context.Background(), projectID, dataset.Name, tableID)
		return resourceLoaded{projectIndex: projectIndex, datasetIndex: datasetIndex, childIndex: childIndex, resource: resource, err: err}
	}
}
