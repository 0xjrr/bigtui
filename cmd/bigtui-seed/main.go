package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/0xjrr/bigtui/internal/auth"
	"github.com/0xjrr/bigtui/internal/seed"
)

func main() {
	projectID := flag.String("project", "", "Google Cloud project ID to seed")
	flag.Parse()
	target := strings.TrimSpace(*projectID)
	if target == "" {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(context.Background(), target); err != nil {
		fmt.Fprintf(os.Stderr, "bigtui-seed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Seeded %s in project %s\n", seed.DatasetID, target)
}

func run(ctx context.Context, projectID string) error {
	if err := auth.EnsureApplicationDefaultCredentials(ctx); err != nil {
		return err
	}
	return seed.Run(ctx, projectID)
}
