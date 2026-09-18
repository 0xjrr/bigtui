package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xjrr/bigtui/internal/auth"
	"github.com/xjrr/bigtui/internal/seed"
)

func main() {
	projectID := flag.String("project", "", "Google Cloud project ID to seed")
	flag.Parse()
	if strings.TrimSpace(*projectID) == "" {
		flag.Usage()
		os.Exit(2)
	}

	ctx := context.Background()
	if err := auth.EnsureApplicationDefaultCredentials(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "bigtui-seed: %v\n", err)
		os.Exit(1)
	}
	if err := seed.Run(ctx, strings.TrimSpace(*projectID)); err != nil {
		fmt.Fprintf(os.Stderr, "bigtui-seed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Seeded %s in project %s\n", seed.DatasetID, strings.TrimSpace(*projectID))
}
