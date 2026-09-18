package project

import (
	"context"
	"fmt"
	"strings"

	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
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
				{Name: "events.raw_events", Kind: "table"},
				{Name: "events.daily_summary", Kind: "view"},
				{Name: "warehouse", Kind: "dataset"},
				{Name: "warehouse.orders", Kind: "table"},
			},
		},
		{
			ID:       "sandbox-reporting",
			Name:     "Sandbox Reporting",
			Location: "EU",
			Resources: []Resource{
				{Name: "finance", Kind: "dataset"},
				{Name: "finance.monthly_revenue", Kind: "view"},
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
	return projects, nil
}
