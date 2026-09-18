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
	Name           string
	Kind           string
	Children       []Resource
	Columns        []string
	Preview        [][]string
	ViewQuery      string
	ExternalSource []string
}

func MockProjects() []Project {
	return []Project{
		{
			ID:       "sandbox-analytics",
			Name:     "Sandbox Analytics",
			Location: "US",
			Resources: []Resource{
				{Name: "events", Kind: "dataset", Children: []Resource{{Name: "customers", Kind: "table", Columns: []string{"customer_id", "name", "segment"}, Preview: [][]string{{"1", "Ada Lovelace", "enterprise"}}}, {Name: "customer_order_totals", Kind: "view", ViewQuery: "SELECT customer_id, name, SUM(amount) AS lifetime_value FROM orders GROUP BY customer_id, name"}, {Name: "partner_feed", Kind: "external", ExternalSource: []string{"gs://partner-feed/events/*.parquet"}}}},
				{Name: "warehouse", Kind: "dataset", Children: []Resource{{Name: "orders", Kind: "table"}}},
			},
		},
		{
			ID:       "sandbox-reporting",
			Name:     "Sandbox Reporting",
			Location: "EU",
			Resources: []Resource{
				{Name: "finance", Kind: "dataset", Children: []Resource{{Name: "monthly_revenue", Kind: "view"}}},
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
			resource := Resource{Name: dataset.DatasetID, Kind: "dataset"}
			tables := dataset.Tables(ctx)
			for {
				table, err := tables.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					client.Close()
					return nil, fmt.Errorf("list tables for dataset %s: %w", dataset.DatasetID, err)
				}
				kind := "table"
				metadata, err := table.Metadata(ctx)
				if err == nil && metadata.ExternalDataConfig != nil {
					kind = "external"
				} else if err == nil && metadata.ViewQuery != "" {
					kind = "view"
				}
				child := Resource{Name: table.TableID, Kind: kind}
				if err == nil {
					child.ViewQuery = metadata.ViewQuery
					if metadata.ExternalDataConfig != nil {
						child.ExternalSource = append(child.ExternalSource, metadata.ExternalDataConfig.SourceURIs...)
					} else {
						for _, field := range metadata.Schema {
							child.Columns = append(child.Columns, field.Name)
						}
						rows := table.Read(ctx)
						for len(child.Preview) < 5 {
							var values []cloudbigquery.Value
							if readErr := rows.Next(&values); readErr != nil {
								break
							}
							preview := make([]string, len(values))
							for valueIndex, value := range values {
								preview[valueIndex] = fmt.Sprint(value)
							}
							child.Preview = append(child.Preview, preview)
						}
					}
				}
				resource.Children = append(resource.Children, child)
			}
			projects[index].Resources = append(projects[index].Resources, resource)
		}
		client.Close()
	}
	return projects, nil
}
