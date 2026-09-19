package bigquery

import (
	"context"
	"fmt"
	"math/big"
	"strings"

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

type Analysis struct {
	Valid          bool
	BytesProcessed int64
	Err            error
}

type Client interface {
	Query(context.Context, string, string) (Result, error)
}

type Analyzer interface {
	Analyze(context.Context, string, string) Analysis
}

type CloudClient struct{}

func (c *CloudClient) Analyze(ctx context.Context, projectID, sql string) Analysis {
	client, err := cloudbigquery.NewClient(ctx, projectID)
	if err != nil {
		return Analysis{Err: err}
	}
	defer client.Close()
	query := client.Query(sql)
	query.DryRun = true
	job, err := query.Run(ctx)
	if err != nil {
		return Analysis{Err: err}
	}
	status := job.LastStatus()
	if err := status.Err(); err != nil {
		return Analysis{Err: err}
	}
	bytes := int64(0)
	if status.Statistics != nil {
		bytes = status.Statistics.TotalBytesProcessed
	}
	return Analysis{Valid: true, BytesProcessed: bytes}
}

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
	if len(it.Schema) > 0 {
		result.Columns = make([]string, len(it.Schema))
		for index, field := range it.Schema {
			result.Columns[index] = field.Name
		}
	}
	for {
		var values []cloudbigquery.Value
		err := it.Next(&values)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return Result{}, err
		}
		if result.Total == 0 && len(result.Columns) == 0 && len(it.Schema) > 0 {
			result.Columns = make([]string, len(it.Schema))
			for index, field := range it.Schema {
				result.Columns[index] = field.Name
			}
		}
		if result.Total == 0 && len(result.Columns) == 0 {
			result.Columns = make([]string, len(values))
			for i := range values {
				result.Columns[i] = fmt.Sprintf("column_%d", i+1)
			}
		}
		row := Row{Values: make([]string, len(values))}
		for i, value := range values {
			row.Values[i] = formatValue(value)
		}
		result.Rows = append(result.Rows, row)
		result.Total++
	}
	return result, nil
}

func formatValue(value cloudbigquery.Value) string {
	switch numeric := value.(type) {
	case *big.Rat:
		return strings.TrimRight(strings.TrimRight(numeric.FloatString(9), "0"), ".")
	case big.Rat:
		return strings.TrimRight(strings.TrimRight(numeric.FloatString(9), "0"), ".")
	default:
		return fmt.Sprint(value)
	}
}
