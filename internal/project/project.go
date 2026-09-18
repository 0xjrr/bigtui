package project

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
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

type gcloudProject struct {
	ProjectID     string `json:"projectId"`
	Name          string `json:"name"`
	ProjectNumber string `json:"projectNumber"`
}

func Load(ctx context.Context) ([]Project, error) {
	command := exec.CommandContext(ctx, "gcloud", "projects", "list", "--format=json")
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list Google Cloud projects: %w", err)
	}
	var listed []gcloudProject
	if err := json.Unmarshal(output, &listed); err != nil {
		return nil, fmt.Errorf("decode Google Cloud projects: %w", err)
	}
	projects := make([]Project, 0, len(listed))
	for _, item := range listed {
		if strings.TrimSpace(item.ProjectID) == "" {
			continue
		}
		projects = append(projects, Project{ID: item.ProjectID, Name: item.Name})
	}
	return projects, nil
}
