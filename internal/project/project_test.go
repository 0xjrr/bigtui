package project

import (
	"strings"
	"testing"
)

func TestMockProjectsExposeResourceDetails(t *testing.T) {
	dataset := MockProjects()[0].Resources[0]
	byKind := map[string]Resource{}
	for _, child := range dataset.Children {
		byKind[child.Kind] = child
	}
	if len(byKind["table"].Preview) == 0 || len(byKind["table"].Columns) == 0 {
		t.Fatalf("mock tables should carry a preview: %#v", byKind["table"])
	}
	if !strings.Contains(byKind["view"].ViewQuery, "SELECT") {
		t.Fatalf("mock views should carry their query: %#v", byKind["view"])
	}
	if len(byKind["external"].ExternalSource) == 0 {
		t.Fatalf("mock external tables should carry their source: %#v", byKind["external"])
	}
	for _, child := range dataset.Children {
		if !child.DetailsLoaded {
			t.Fatalf("mock children should be fully loaded: %#v", child)
		}
	}
}

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

func TestMockProjectsContainTableKinds(t *testing.T) {
	seen := map[string]bool{}
	for _, project := range MockProjects() {
		for _, dataset := range project.Resources {
			for _, child := range dataset.Children {
				seen[child.Kind] = true
			}
		}
	}
	for _, kind := range []string{"table", "external", "view"} {
		if !seen[kind] {
			t.Fatalf("mock catalog is missing %s resources", kind)
		}
	}
}

func TestTableKindMapsListResponseTypes(t *testing.T) {
	tests := map[string]string{
		"TABLE":             "table",
		"VIEW":              "view",
		"MATERIALIZED_VIEW": "view",
		"EXTERNAL":          "external",
		"SNAPSHOT":          "table",
	}
	for tableType, want := range tests {
		if got := tableKind(tableType); got != want {
			t.Errorf("tableKind(%q) = %q, want %q", tableType, got, want)
		}
	}
}
