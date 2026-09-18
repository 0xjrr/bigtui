package project

import (
	"context"
	"fmt"
	"strings"

	cloudbigquery "cloud.google.com/go/bigquery"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iterator"
)

type Project struct {
	ID        string
	Name      string
	Location  string
	Resources []Resource
}

type Resource struct {
	Name string
	Kind string
}

func MockProjects() []Project {
	return []Project{
		{
			ID:       "sandbox-analytics",
			Name:     "Sandbox Analytics",
			Location: "US",
			Resources: []Resource{
				{Name: "events", Kind: "dataset"},
				{Name: "warehouse", Kind: "dataset"},
			},
		},
		{
			ID:       "sandbox-reporting",
			Name:     "Sandbox Reporting",
			Location: "EU",
			Resources: []Resource{
				{Name: "finance", Kind: "dataset"},
			},
		},
	}
}

func Load(ctx context.Context) ([]Project, error) {
	service, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Google Cloud project service: %w", err)
	}
	response, err := service.Projects.List().Do()
	if err != nil {
		return nil, fmt.Errorf("list Google Cloud projects: %w", err)
	}
	projects := make([]Project, 0, len(response.Projects))
	for _, item := range response.Projects {
		if strings.TrimSpace(item.ProjectId) == "" {
			continue
		}
		projects = append(projects, Project{ID: item.ProjectId, Name: item.Name})
	}
	for index := range projects {
		client, err := cloudbigquery.NewClient(ctx, projects[index].ID)
		if err != nil {
			return nil, fmt.Errorf("connect to project %s: %w", projects[index].ID, err)
		}
		it := client.Datasets(ctx)
		for {
			dataset, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				client.Close()
				return nil, fmt.Errorf("list datasets for project %s: %w", projects[index].ID, err)
			}
			projects[index].Resources = append(projects[index].Resources, Resource{Name: dataset.DatasetID, Kind: "dataset"})
		}
		client.Close()
	}
	return projects, nil
}
