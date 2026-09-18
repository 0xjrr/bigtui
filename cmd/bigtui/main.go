package main

import (
	"context"
	"fmt"
	"os"

	"github.com/xjrr/bigtui/internal/app"
	"github.com/xjrr/bigtui/internal/auth"
	"github.com/xjrr/bigtui/internal/bigquery"
)

func main() {
	ctx := context.Background()
	if err := auth.EnsureApplicationDefaultCredentials(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "bigtui: %v\n", err)
		os.Exit(1)
	}
	client, err := bigquery.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bigtui: %v\n", err)
		os.Exit(1)
	}

	program := app.New(client)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "bigtui: %v\n", err)
		os.Exit(1)
	}
}
