package seed

import (
	"context"
	"fmt"

	cloudbigquery "cloud.google.com/go/bigquery"
)

const DatasetID = "bigtui_demo"

func Run(ctx context.Context, projectID string) error {
	client, err := cloudbigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("connect to project %s: %w", projectID, err)
	}
	defer client.Close()

	statements := []string{
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS `%s.%s` OPTIONS(location = 'US')", projectID, DatasetID),
		fmt.Sprintf("CREATE OR REPLACE TABLE `%s.%s.customers` (customer_id INT64, name STRING, segment STRING)", projectID, DatasetID),
		fmt.Sprintf("CREATE OR REPLACE TABLE `%s.%s.orders` (order_id INT64, customer_id INT64, amount NUMERIC, ordered_at DATE)", projectID, DatasetID),
		fmt.Sprintf("INSERT INTO `%s.%s.customers` (customer_id, name, segment) VALUES (1, 'Ada Lovelace', 'enterprise'), (2, 'Grace Hopper', 'startup'), (3, 'Katherine Johnson', 'enterprise')", projectID, DatasetID),
		fmt.Sprintf("INSERT INTO `%s.%s.orders` (order_id, customer_id, amount, ordered_at) VALUES (1001, 1, 1200.50, DATE '2026-01-15'), (1002, 1, 800.00, DATE '2026-02-02'), (1003, 2, 450.25, DATE '2026-02-18'), (1004, 3, 2100.00, DATE '2026-03-04')", projectID, DatasetID),
		fmt.Sprintf("CREATE OR REPLACE VIEW `%s.%s.customer_order_totals` AS SELECT c.customer_id, c.name, c.segment, COUNT(o.order_id) AS order_count, SUM(o.amount) AS lifetime_value FROM `%s.%s.customers` c LEFT JOIN `%s.%s.orders` o USING (customer_id) GROUP BY c.customer_id, c.name, c.segment", projectID, DatasetID, projectID, DatasetID, projectID, DatasetID),
	}

	for _, statement := range statements {
		job, err := client.Query(statement).Run(ctx)
		if err != nil {
			return fmt.Errorf("run seed statement: %w", err)
		}
		if _, err := job.Wait(ctx); err != nil {
			return fmt.Errorf("wait for seed statement: %w", err)
		}
	}
	return nil
}
