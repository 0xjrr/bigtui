package project

import (
	"context"
	"fmt"
	"strings"
	"time"

	cloudbigquery "cloud.google.com/go/bigquery"
	cloudbigqueryapi "google.golang.org/api/bigquery/v2"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
)

const listPageSize = 1000

type Project struct {
	ID        string
	Name      string
	Location  string
	Resources []Resource
}

type Resource struct {
	Name                   string
	Kind                   string
	Children               []Resource
	ChildrenLoaded         bool
	Columns                []string
	Preview                [][]string
	ViewQuery              string
	ExternalSource         []string
	ExternalFormat         string
	ID                     string
	Created                time.Time
	Modified               time.Time
	Expiration             time.Time
	Location               string
	Description            string
	Labels                 map[string]string
	LegacySQL              bool
	DetailsLoaded          bool
	NumRows                uint64
	NumBytes               int64
	LongTermBytes          int64
	PartitionType          string
	PartitionField         string
	PartitionExpiration    time.Duration
	RequirePartitionFilter bool
	Clustering             []string
}

func MockProjects() []Project {
	return []Project{
		{
			ID:       "sandbox-analytics",
			Name:     "Sandbox Analytics",
			Location: "US",
			Resources: []Resource{
				{Name: "events", Kind: "dataset", Children: []Resource{{Name: "customers", Kind: "table", DetailsLoaded: true, Columns: []string{"customer_id", "name", "segment"}, Preview: [][]string{{"1", "Ada Lovelace", "enterprise"}}}, {Name: "customer_order_totals", Kind: "view", DetailsLoaded: true, ViewQuery: "SELECT customer_id, name, SUM(amount) AS lifetime_value FROM orders GROUP BY customer_id, name"}, {Name: "partner_feed", Kind: "external", DetailsLoaded: true, ExternalSource: []string{"gs://partner-feed/events/*.parquet"}}}},
				{Name: "warehouse", Kind: "dataset", Children: []Resource{{Name: "orders", Kind: "table", DetailsLoaded: true}}},
			},
		},
		{
			ID:       "sandbox-reporting",
			Name:     "Sandbox Reporting",
			Location: "EU",
			Resources: []Resource{
				{Name: "finance", Kind: "dataset", Children: []Resource{{Name: "monthly_revenue", Kind: "view", DetailsLoaded: true}}},
			},
		},
	}
}

type ResourceLoader interface {
	LoadResource(context.Context, string, string, string) (Resource, error)
}

type CatalogLoader interface {
	ResourceLoader
	Load(context.Context) ([]Project, error)
	LoadDatasets(context.Context, string, bool) ([]Resource, error)
	LoadTables(context.Context, string, string) ([]Resource, error)
}

type Loader struct {
	service *cloudbigqueryapi.Service
}

func NewLoader(ctx context.Context) (*Loader, error) {
	service, err := cloudbigqueryapi.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create BigQuery service: %w", err)
	}
	return &Loader{service: service}, nil
}

func Load(ctx context.Context) ([]Project, error) {
	loader, err := NewLoader(ctx)
	if err != nil {
		return nil, err
	}
	return loader.Load(ctx)
}

func (l *Loader) Load(ctx context.Context) ([]Project, error) {
	service, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Google Cloud project service: %w", err)
	}
	projects := []Project{}
	pageToken := ""
	for {
		call := service.Projects.List()
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		response, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list Google Cloud projects: %w", err)
		}
		for _, item := range response.Projects {
			if strings.TrimSpace(item.ProjectId) == "" {
				continue
			}
			projects = append(projects, Project{ID: item.ProjectId, Name: item.Name})
		}
		pageToken = response.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return projects, nil
}

func (l *Loader) LoadDatasets(ctx context.Context, projectID string, includeHidden bool) ([]Resource, error) {
	datasets, err := l.listDatasets(ctx, projectID, includeHidden)
	if err != nil {
		return nil, err
	}
	resources := make([]Resource, 0, len(datasets))
	for _, dataset := range datasets {
		resources = append(resources, Resource{Name: dataset.DatasetReference.DatasetId, Kind: "dataset"})
	}
	return resources, nil
}

func (l *Loader) LoadTables(ctx context.Context, projectID, datasetID string) ([]Resource, error) {
	tables, err := l.listTables(ctx, projectID, datasetID)
	if err != nil {
		return nil, err
	}
	resources := make([]Resource, 0, len(tables))
	for _, table := range tables {
		resources = append(resources, Resource{Name: table.TableReference.TableId, Kind: tableKind(table.Type)})
	}
	return resources, nil
}

func (l *Loader) listDatasets(ctx context.Context, projectID string, includeHidden bool) ([]*cloudbigqueryapi.DatasetListDatasets, error) {
	datasets := []*cloudbigqueryapi.DatasetListDatasets{}
	pageToken := ""
	for {
		call := l.service.Datasets.List(projectID).All(includeHidden).MaxResults(listPageSize).Context(ctx)
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		response, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list datasets for project %s: %w", projectID, err)
		}
		datasets = append(datasets, response.Datasets...)
		pageToken = response.NextPageToken
		if pageToken == "" {
			return datasets, nil
		}
	}
}

func (l *Loader) listTables(ctx context.Context, projectID, datasetID string) ([]*cloudbigqueryapi.TableListTables, error) {
	tables := []*cloudbigqueryapi.TableListTables{}
	pageToken := ""
	for {
		call := l.service.Tables.List(projectID, datasetID).MaxResults(listPageSize).Context(ctx)
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		response, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list tables for dataset %s: %w", datasetID, err)
		}
		tables = append(tables, response.Tables...)
		pageToken = response.NextPageToken
		if pageToken == "" {
			return tables, nil
		}
	}
}

func tableKind(tableType string) string {
	switch tableType {
	case "VIEW", "MATERIALIZED_VIEW":
		return "view"
	case "EXTERNAL":
		return "external"
	default:
		return "table"
	}
}

func (l *Loader) LoadResource(ctx context.Context, projectID, datasetID, tableID string) (Resource, error) {
	client, err := cloudbigquery.NewClient(ctx, projectID)
	if err != nil {
		return Resource{}, fmt.Errorf("connect to project %s: %w", projectID, err)
	}
	defer client.Close()
	table := client.Dataset(datasetID).Table(tableID)
	metadata, err := table.Metadata(ctx)
	if err != nil {
		return Resource{}, fmt.Errorf("load metadata for %s.%s: %w", datasetID, tableID, err)
	}
	resource := Resource{
		Name: tableID, Kind: "table", ID: metadata.FullID, DetailsLoaded: true,
		Created: metadata.CreationTime, Modified: metadata.LastModifiedTime,
		Expiration: metadata.ExpirationTime, Location: metadata.Location,
		Description: metadata.Description, Labels: metadata.Labels,
		NumRows: metadata.NumRows, NumBytes: metadata.NumBytes, LongTermBytes: metadata.NumLongTermBytes,
		RequirePartitionFilter: metadata.RequirePartitionFilter, ViewQuery: metadata.ViewQuery,
		LegacySQL: metadata.UseLegacySQL,
	}
	if metadata.ExternalDataConfig != nil {
		resource.Kind = "external"
		resource.ExternalFormat = string(metadata.ExternalDataConfig.SourceFormat)
		resource.ExternalSource = append(resource.ExternalSource, metadata.ExternalDataConfig.SourceURIs...)
	} else {
		if metadata.ViewQuery != "" {
			resource.Kind = "view"
		}
		for _, field := range metadata.Schema {
			resource.Columns = append(resource.Columns, field.Name)
		}
		rows := table.Read(ctx)
		for len(resource.Preview) < 5 {
			var values []cloudbigquery.Value
			if readErr := rows.Next(&values); readErr != nil {
				break
			}
			preview := make([]string, len(values))
			for valueIndex, value := range values {
				preview[valueIndex] = fmt.Sprint(value)
			}
			resource.Preview = append(resource.Preview, preview)
		}
	}
	if metadata.Clustering != nil {
		resource.Clustering = append(resource.Clustering, metadata.Clustering.Fields...)
	}
	if metadata.TimePartitioning != nil {
		resource.PartitionType = string(metadata.TimePartitioning.Type)
		resource.PartitionField = metadata.TimePartitioning.Field
		resource.PartitionExpiration = metadata.TimePartitioning.Expiration
	} else if metadata.RangePartitioning != nil {
		resource.PartitionType = "RANGE"
		resource.PartitionField = metadata.RangePartitioning.Field
	}
	return resource, nil
}
