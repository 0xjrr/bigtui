package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/xjrr/bigtui/internal/format"
	"github.com/xjrr/bigtui/internal/project"
	"github.com/xjrr/bigtui/internal/ui/grid"
	"github.com/xjrr/bigtui/internal/ui/text"
	"github.com/xjrr/bigtui/internal/ui/theme"
)

func (m model) infoView() string {
	outerWidth := max(40, m.width-4)
	outerHeight := max(12, m.height-4)
	styleWidth := max(1, outerWidth-2)
	styleHeight := max(1, outerHeight-2)
	lines := text.WrapLines(m.infoLines(), max(20, styleWidth-6))
	visible := scrollWindow(lines, m.infoScroll, max(1, max(1, styleHeight-6)-1))
	visible = append(visible, "", theme.Bright("Up/Down scroll  ·  Enter/Esc close"))
	return theme.Modal(styleWidth, styleHeight).Render(lipgloss.JoinVertical(lipgloss.Left, visible...))
}

func scrollWindow(lines []string, scroll, viewport int) []string {
	start := clamp(scroll, 0, max(0, len(lines)-viewport))
	end := min(len(lines), start+viewport)
	return append([]string{}, lines[start:end]...)
}

func (m model) infoLines() []string {
	title, name, details := m.infoContent()
	lines := []string{theme.Title(title), "", theme.Strong(name), ""}
	for _, detail := range details {
		lines = append(lines, theme.Dim(detail))
	}
	return lines
}

func (m model) infoContent() (string, string, []string) {
	item := m.projects[m.active]
	if !m.datasetExists(m.active, m.selectedDataset) {
		name := item.Name
		if name == "" {
			name = item.ID
		}
		return "PROJECT", name, []string{"Project ID  " + item.ID}
	}
	dataset := item.Resources[m.selectedDataset]
	if m.selectedChild < 0 || m.selectedChild >= len(dataset.Children) {
		return "DATASET", dataset.Name, []string{"Project  " + item.ID, fmt.Sprintf("Children  %d tables/views", len(dataset.Children))}
	}
	child := dataset.Children[m.selectedChild]
	return strings.ToUpper(child.Kind), child.Name, m.childInfoLines(item.ID, dataset.Name, child)
}

func (m model) childInfoLines(projectID, datasetName string, child project.Resource) []string {
	details := []string{"Project  " + projectID, "Dataset  " + datasetName, "Type     " + child.Kind}
	if !child.DetailsLoaded {
		return append(details, "Details  Loading...")
	}
	details = append(details, resourceInfoLines(child)...)
	details = append(details, m.viewQueryLines(child)...)
	details = append(details, externalSourceLines(child)...)
	details = append(details, previewInfoLines(child)...)
	return details
}

func (m model) viewQueryLines(child project.Resource) []string {
	if len(child.ViewQuery) == 0 {
		return nil
	}
	lines := []string{"", "Query"}
	for _, line := range text.Wrap(child.ViewQuery, max(20, m.width-10)) {
		lines = append(lines, "  "+line)
	}
	return lines
}

func externalSourceLines(child project.Resource) []string {
	if len(child.ExternalSource) == 0 {
		return nil
	}
	lines := []string{"", "External source"}
	for _, source := range child.ExternalSource {
		lines = append(lines, "  "+text.Truncate(source, 58))
	}
	return lines
}

func previewInfoLines(child project.Resource) []string {
	if len(child.Columns) == 0 {
		return nil
	}
	return append([]string{"", "Preview"}, grid.Preview(child.Columns, child.Preview)...)
}

func resourceInfoLines(resource project.Resource) []string {
	lines := []string{"", "Table info"}
	if resource.ID != "" {
		lines = append(lines, "  Table ID       "+resource.ID)
	}
	if !resource.Created.IsZero() {
		lines = append(lines, "  Created        "+resource.Created.Format(time.RFC3339))
	}
	if !resource.Modified.IsZero() {
		lines = append(lines, "  Last modified  "+resource.Modified.Format(time.RFC3339))
	}
	lines = append(lines, "  Expiration     "+expirationLabel(resource.Expiration))
	if resource.Location != "" {
		lines = append(lines, "  Data location  "+resource.Location)
	}
	lines = append(lines, fmt.Sprintf("  Legacy SQL     %t", resource.LegacySQL))
	if resource.Description != "" {
		lines = append(lines, "  Description    "+resource.Description)
	}
	if len(resource.Labels) > 0 {
		lines = append(lines, "  Labels         "+format.Labels(resource.Labels))
	}
	lines = append(lines, storageInfoLines(resource)...)
	lines = append(lines, partitioningLines(resource)...)
	lines = append(lines, clusteringLines(resource)...)
	return lines
}

func storageInfoLines(resource project.Resource) []string {
	if resource.Kind != "table" && resource.Kind != "external" {
		return nil
	}
	return []string{
		"",
		"Storage info",
		fmt.Sprintf("  Number of rows          %d", resource.NumRows),
		"  Total logical bytes     " + format.Bytes(resource.NumBytes),
		"  Long term logical bytes " + format.Bytes(resource.LongTermBytes),
	}
}

func partitioningLines(resource project.Resource) []string {
	if resource.PartitionType == "" && resource.PartitionField == "" {
		return nil
	}
	lines := []string{"", "Partitioning"}
	if resource.PartitionType != "" {
		lines = append(lines, "  Type             "+resource.PartitionType)
	}
	if resource.PartitionField != "" {
		lines = append(lines, "  Field            "+resource.PartitionField)
	}
	if resource.PartitionExpiration > 0 {
		lines = append(lines, "  Partition expiry "+resource.PartitionExpiration.String())
	}
	return append(lines, fmt.Sprintf("  Require filter   %t", resource.RequirePartitionFilter))
}

func clusteringLines(resource project.Resource) []string {
	if len(resource.Clustering) == 0 {
		return nil
	}
	return []string{"", "Clustering", "  Fields           " + strings.Join(resource.Clustering, ", ")}
}

func expirationLabel(expiration time.Time) string {
	if expiration.IsZero() {
		return "NEVER"
	}
	return expiration.Format(time.RFC3339)
}
