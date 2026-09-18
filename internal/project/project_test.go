package project

import "testing"

func TestMockProjectsContainResourceKinds(t *testing.T) {
	projects := MockProjects()
	if len(projects) != 2 {
		t.Fatalf("expected two mock projects: %#v", projects)
	}
	seen := map[string]bool{}
	for _, project := range projects {
		for _, resource := range project.Resources {
			seen[resource.Kind] = true
		}
	}
	for _, kind := range []string{"dataset", "table", "view"} {
		if !seen[kind] {
			t.Fatalf("mock catalog is missing %s resources", kind)
		}
	}
}
