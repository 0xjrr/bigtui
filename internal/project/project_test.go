package project

import "testing"

func TestMockProjectsContainDatasets(t *testing.T) {
	projects := MockProjects()
	if len(projects) != 2 {
		t.Fatalf("expected two mock projects: %#v", projects)
	}
	seen := false
	for _, project := range projects {
		for _, resource := range project.Resources {
			if resource.Kind == "dataset" {
				seen = true
			}
		}
	}
	if !seen {
		t.Fatal("mock catalog is missing datasets")
	}
}
