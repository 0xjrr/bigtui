package bigquery

import (
	"context"
	"fmt"

	cloudbigquery "cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

type Row struct {
	Values []string
}

type Result struct {
	Columns []string
	Rows    []Row
	Total   uint64
}

type Client interface {
	Query(context.Context, string, string) (Result, error)
}

type CloudClient struct{}

func NewClient(ctx context.Context) (Client, error) {
	return &CloudClient{}, nil
}

func (c *CloudClient) Query(ctx context.Context, projectID, sql string) (Result, error) {
	client, err := cloudbigquery.NewClient(ctx, projectID)
	if err != nil {
		return Result{}, err
	}
	defer client.Close()

	query := client.Query(sql)
	it, err := query.Read(ctx)
	if err != nil {
		return Result{}, err
	}

	result := Result{}
	for {
		var values []cloudbigquery.Value
		err := it.Next(&values)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return Result{}, err
		}
		if result.Total == 0 {
			result.Columns = make([]string, len(values))
			for i := range values {
				result.Columns[i] = fmt.Sprintf("column_%d", i+1)
			}
		}
		row := Row{Values: make([]string, len(values))}
		for i, value := range values {
			row.Values[i] = fmt.Sprint(value)
		}
		result.Rows = append(result.Rows, row)
		result.Total++
	}
	return result, nil
}
