package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/xjrr/bigtui/internal/app"
	"github.com/xjrr/bigtui/internal/auth"
	"github.com/xjrr/bigtui/internal/bigquery"
	"github.com/xjrr/bigtui/internal/project"
)

func main() {
	mock := flag.Bool("mock", false, "load local fixture projects, datasets, tables, and views")
	flag.Parse()

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

	projects := project.MockProjects()
	if !*mock {
		projects, err = project.Load(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bigtui: %v\n", err)
			os.Exit(1)
		}
	}
	program := app.NewWithProjects(client, projects)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "bigtui: %v\n", err)
		os.Exit(1)
	}
}
