package completion

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"github.com/xjrr/bigtui/internal/project"
)

type Request struct {
	Project string
	SQL     string
	Cursor  int
}

type Item struct {
	Label      string
	Detail     string
	InsertText string
}

type Provider interface {
	Complete(context.Context, Request) ([]Item, error)
}

type Chain struct {
	Providers []Provider
}

func (c Chain) Complete(ctx context.Context, request Request) ([]Item, error) {
	for _, provider := range c.Providers {
		items, err := provider.Complete(ctx, request)
		if err != nil {
			continue
		}
		if len(items) > 0 {
			return items, nil
		}
	}
	return nil, nil
}

func WordPrefix(sql string, cursor int) string {
	runes := []rune(sql)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	start := cursor
	for start > 0 && isWordRune(runes[start-1]) {
		start--
	}
	return string(runes[start:cursor])
}

func ReferenceParts(sql string, cursor int) []string {
	runes := []rune(sql)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	start := cursor
	for start > 0 && isReferenceRune(runes[start-1]) {
		start--
	}
	token := strings.Trim(string(runes[start:cursor]), "`")
	if token == "" {
		return nil
	}
	parts := strings.Split(token, ".")
	for index := range parts {
		parts[index] = strings.Trim(parts[index], "`")
	}
	return parts
}

func ReferenceIsQuoted(sql string, cursor int) bool {
	runes := []rune(sql)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	start := cursor
	for start > 0 && isReferenceRune(runes[start-1]) {
		start--
	}
	return start < len(runes) && runes[start] == '`'
}

func IsResourceContext(sql string, cursor int) bool {
	if len(ReferenceParts(sql, cursor)) > 1 {
		return true
	}
	runes := []rune(sql)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	fields := strings.Fields(string(runes[:cursor]))
	if len(fields) < 2 {
		return false
	}
	switch strings.ToUpper(strings.Trim(fields[len(fields)-2], "`(),")) {
	case "FROM", "JOIN", "INTO", "UPDATE", "TABLE", "VIEW":
		return true
	default:
		return false
	}
}

func isReferenceRune(r rune) bool {
	return r == '`' || r == '.' || r == '-' || isWordRune(r)
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

type KeywordProvider struct{}

func (KeywordProvider) Complete(_ context.Context, request Request) ([]Item, error) {
	prefix := strings.ToUpper(WordPrefix(request.SQL, request.Cursor))
	if prefix == "" {
		return nil, nil
	}
	items := []Item{}
	for _, snippet := range sqlSnippetMappings {
		if prefix == snippet.Prefix {
			items = append(items, Item{Label: snippet.Label, Detail: snippet.Description, InsertText: snippet.InsertText})
		}
	}
	for _, keyword := range sqlKeywordMappings {
		if strings.HasPrefix(keyword, prefix) {
			items = append(items, Item{Label: keyword, Detail: "keyword", InsertText: keyword})
		}
	}
	return items, nil
}

type FunctionProvider struct{}

func (FunctionProvider) Complete(_ context.Context, request Request) ([]Item, error) {
	prefix := strings.ToUpper(WordPrefix(request.SQL, request.Cursor))
	if prefix == "" {
		return nil, nil
	}
	items := []Item{}
	for _, function := range sqlFunctionMappings {
		if strings.HasPrefix(function.Name, prefix) {
			items = append(items, Item{Label: function.Name, Detail: function.Description, InsertText: function.Name})
		}
	}
	return items, nil
}

type CatalogProvider struct {
	Catalog []project.Project
}

func (p CatalogProvider) Complete(_ context.Context, request Request) ([]Item, error) {
	prefix := strings.ToLower(WordPrefix(request.SQL, request.Cursor))
	parts := ReferenceParts(request.SQL, request.Cursor)
	hasQualifier := len(parts) > 1
	qualifiedReference := len(parts) > 1 && parts[len(parts)-1] == ""
	tableContext := false
	targetDataset := ""
	targetProject := request.Project
	if len(parts) >= 3 {
		tableContext = true
		targetDataset = parts[1]
		targetProject = parts[0]
	} else if len(parts) == 2 && parts[0] != request.Project {
		for _, proj := range p.Catalog {
			if proj.ID != request.Project {
				continue
			}
			for _, dataset := range proj.Resources {
				if dataset.Kind == "dataset" && dataset.Name == parts[0] {
					tableContext = true
					targetDataset = parts[0]
				}
			}
		}
	}
	if prefix == "" && !qualifiedReference {
		return nil, nil
	}
	items := []Item{}
	seen := map[string]bool{}
	add := func(label, detail string) {
		if !strings.HasPrefix(strings.ToLower(label), prefix) || seen[label] {
			return
		}
		seen[label] = true
		items = append(items, Item{Label: label, Detail: detail, InsertText: label})
	}
	for _, proj := range p.Catalog {
		if hasQualifier && !tableContext && proj.ID != parts[0] {
			continue
		}
		if tableContext && targetProject != "" && proj.ID != targetProject {
			continue
		}
		if !hasQualifier {
			add(proj.ID, "project")
		}
		for _, dataset := range proj.Resources {
			if dataset.Kind != "dataset" {
				continue
			}
			if tableContext && dataset.Name != targetDataset {
				continue
			}
			if !hasQualifier {
				add(dataset.Name, "dataset · "+proj.ID)
			}
			if hasQualifier && !tableContext {
				add(dataset.Name, "dataset · "+proj.ID)
				continue
			}
			for _, child := range dataset.Children {
				add(child.Name, child.Kind+" · "+dataset.Name)
				for _, column := range child.Columns {
					add(column, "column · "+child.Name)
				}
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	return items, nil
}

func Merge(sets ...[]Item) []Item {
	seen := map[string]bool{}
	merged := []Item{}
	for _, set := range sets {
		for _, item := range set {
			if seen[item.Label] {
				continue
			}
			seen[item.Label] = true
			merged = append(merged, item)
		}
	}
	return merged
}
