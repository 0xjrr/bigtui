package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/0xjrr/bigtui/internal/app"
	"github.com/0xjrr/bigtui/internal/auth"
	"github.com/0xjrr/bigtui/internal/bigquery"
	"github.com/0xjrr/bigtui/internal/project"
)

func main() {
	mock := flag.Bool("mock", false, "load local fixture projects, datasets, tables, and views")
	flag.Parse()

	if err := run(context.Background(), *mock); err != nil {
		fmt.Fprintf(os.Stderr, "bigtui: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, mock bool) error {
	if err := auth.EnsureApplicationDefaultCredentials(ctx); err != nil {
		return err
	}
	client, err := bigquery.NewClient(ctx)
	if err != nil {
		return err
	}
	program, err := newProgram(ctx, client, mock)
	if err != nil {
		return err
	}
	_, err = program.Run()
	return err
}

func newProgram(ctx context.Context, client bigquery.Client, mock bool) (*tea.Program, error) {
	if mock {
		return app.NewWithProjects(client, project.MockProjects()), nil
	}
	loader, err := project.NewLoader(ctx)
	if err != nil {
		return nil, err
	}
	return app.NewWithLoader(client, loader), nil
}
