package completion

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xjrr/bigtui/internal/project"
)

type providerFunc func(context.Context, Request) ([]Item, error)

func (f providerFunc) Complete(ctx context.Context, request Request) ([]Item, error) {
	return f(ctx, request)
}

func TestChainUsesFirstProviderWithResults(t *testing.T) {
	chain := Chain{Providers: []Provider{
		providerFunc(func(context.Context, Request) ([]Item, error) { return nil, nil }),
		providerFunc(func(context.Context, Request) ([]Item, error) { return []Item{{Label: "orders"}}, nil }),
	}}
	items, err := chain.Complete(context.Background(), Request{Project: "demo"})
	if err != nil || len(items) != 1 || items[0].Label != "orders" {
		t.Fatalf("unexpected completion: %#v, %v", items, err)
	}
}

func TestWordPrefixExtractsIdentifierBeforeCursor(t *testing.T) {
	cases := map[string]struct {
		sql    string
		cursor int
		want   string
	}{
		"mid identifier":  {sql: "SELECT * FROM cust", cursor: 19, want: "cust"},
		"after space":     {sql: "SELECT * FROM ", cursor: 14, want: ""},
		"cursor at start": {sql: "orders", cursor: 0, want: ""},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			if got := WordPrefix(testCase.sql, testCase.cursor); got != testCase.want {
				t.Fatalf("WordPrefix(%q, %d) = %q, want %q", testCase.sql, testCase.cursor, got, testCase.want)
			}
		})
	}
}

func TestReferencePartsSplitsQualifiedBigQueryNames(t *testing.T) {
	cases := map[string][]string{
		"SELECT * FROM `demo-project.customers.cus":     {"demo-project", "customers", "cus"},
		"SELECT * FROM `demo-project`.`customers`.`cus": {"demo-project", "customers", "cus"},
		"SELECT * FROM `demo-project`.`customers`.":     {"demo-project", "customers", ""},
	}
	for sql, want := range cases {
		parts := ReferenceParts(sql, len([]rune(sql)))
		if len(parts) != len(want) {
			t.Fatalf("ReferenceParts(%q) returned %#v, want %#v", sql, parts, want)
		}
		for index := range want {
			if parts[index] != want[index] {
				t.Fatalf("ReferenceParts(%q) returned %#v, want %#v", sql, parts, want)
			}
		}
	}
}

func TestReferenceIsQuotedDetectsQualifiedBackticks(t *testing.T) {
	quoted := "FROM `demo-project`.`customers`.`profiles"
	if !ReferenceIsQuoted(quoted, len([]rune(quoted))) {
		t.Fatal("expected separately quoted reference to be detected")
	}
	unquoted := "FROM demo-project.customers.profiles"
	if ReferenceIsQuoted(unquoted, len([]rune(unquoted))) {
		t.Fatal("did not expect unquoted reference to be detected")
	}
}

func TestIsResourceContextOnlyMatchesResourceClauses(t *testing.T) {
	tests := map[string]bool{
		"SELECT cu":                  false,
		"SELECT * FROM cu":           true,
		"SELECT * JOIN cu":           true,
		"SELECT * FROM `demo.cu":     true,
		"SELECT * WHERE cu":          false,
		"SELECT * FROM customers.cu": true,
	}
	for sql, want := range tests {
		if got := IsResourceContext(sql, len([]rune(sql))); got != want {
			t.Errorf("IsResourceContext(%q) = %v, want %v", sql, got, want)
		}
	}
}

func TestKeywordProviderFiltersByPrefix(t *testing.T) {
	items, err := KeywordProvider{}.Complete(context.Background(), Request{SQL: "sel", Cursor: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, item := range items {
		if item.Label != "SELECT" {
			t.Fatalf("unexpected keyword for prefix 'sel': %q", item.Label)
		}
		found = true
	}
	if !found {
		t.Fatal("expected SELECT to match prefix 'sel'")
	}
}

func TestKeywordProviderSuggestsSFQuerySnippet(t *testing.T) {
	items, err := KeywordProvider{}.Complete(context.Background(), Request{SQL: "sf", Cursor: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) == 0 || items[0].Label != "SELECT * FROM `" || items[0].InsertText != "SELECT * FROM `" || items[0].Detail != "query snippet" {
		t.Fatalf("unexpected sf snippet completion: %#v", items)
	}
}

func TestKeywordProviderReturnsNoneWithoutPrefix(t *testing.T) {
	items, err := KeywordProvider{}.Complete(context.Background(), Request{SQL: "SELECT * FROM t ", Cursor: 16})
	if err != nil || len(items) != 0 {
		t.Fatalf("expected no keyword suggestions without a prefix: %#v, %v", items, err)
	}
}

func TestFunctionProviderUsesOfficialNamesAndDescriptions(t *testing.T) {
	items, err := FunctionProvider{}.Complete(context.Background(), Request{SQL: "approx_", Cursor: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected official BigQuery functions for prefix approx_")
	}
	for _, item := range items {
		if !strings.HasPrefix(item.Label, "APPROX_") || item.Detail == "" {
			t.Fatalf("unexpected function completion: %#v", item)
		}
	}
}

func TestFunctionCatalogMatchesOfficialExtraction(t *testing.T) {
	if len(sqlFunctionMappings) != 401 {
		t.Fatalf("expected 401 callable function mappings from the official reference, got %d", len(sqlFunctionMappings))
	}
	for _, function := range sqlFunctionMappings {
		if function.Name == "" || function.Description == "" {
			t.Fatalf("function mapping is incomplete: %#v", function)
		}
	}
}

func TestCatalogProviderSuggestsAcrossCatalogLevels(t *testing.T) {
	catalog := []project.Project{
		{
			ID: "demo-project",
			Resources: []project.Resource{
				{Name: "customers", Kind: "dataset", Children: []project.Resource{
					{Name: "customer_profiles", Kind: "table", Columns: []string{"customer_id", "customer_name"}},
				}},
			},
		},
	}
	provider := CatalogProvider{Catalog: catalog}
	items, err := provider.Complete(context.Background(), Request{SQL: "SELECT cust", Cursor: 11})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	labels := map[string]string{}
	for _, item := range items {
		labels[item.Label] = item.Detail
	}
	if _, ok := labels["customers"]; !ok {
		t.Fatalf("expected dataset suggestion: %#v", items)
	}
	if _, ok := labels["customer_profiles"]; !ok {
		t.Fatalf("expected table suggestion: %#v", items)
	}
	if _, ok := labels["customer_id"]; !ok {
		t.Fatalf("expected column suggestion: %#v", items)
	}
	if _, ok := labels["customer_name"]; !ok {
		t.Fatalf("expected column suggestion: %#v", items)
	}
}

func TestCatalogProviderScopesQualifiedCompletionStages(t *testing.T) {
	provider := CatalogProvider{Catalog: []project.Project{{
		ID:        "demo-project",
		Resources: []project.Resource{{Name: "customers", Kind: "dataset", Children: []project.Resource{{Name: "profiles", Kind: "table"}}}},
	}}}
	datasetItems, err := provider.Complete(context.Background(), Request{Project: "demo-project", SQL: "FROM demo-project.", Cursor: len("FROM demo-project.")})
	if err != nil || len(datasetItems) != 1 || datasetItems[0].Label != "customers" {
		t.Fatalf("project-dot completion should contain only datasets: %#v, %v", datasetItems, err)
	}
	quotedDatasetItems, err := provider.Complete(context.Background(), Request{Project: "demo-project", SQL: "FROM `demo-project`.", Cursor: len("FROM `demo-project`.")})
	if err != nil || len(quotedDatasetItems) != 1 || quotedDatasetItems[0].Label != "customers" {
		t.Fatalf("separately quoted project-dot completion should contain only datasets: %#v, %v", quotedDatasetItems, err)
	}
	tableItems, err := provider.Complete(context.Background(), Request{Project: "demo-project", SQL: "FROM demo-project.customers.", Cursor: len("FROM demo-project.customers.")})
	if err != nil || len(tableItems) != 1 || tableItems[0].Label != "profiles" {
		t.Fatalf("dataset-dot completion should contain only tables/views: %#v, %v", tableItems, err)
	}
	quotedItems, err := provider.Complete(context.Background(), Request{Project: "demo-project", SQL: "FROM `demo-project`.`customers`.", Cursor: len("FROM `demo-project`.`customers`.")})
	if err != nil || len(quotedItems) != 1 || quotedItems[0].Label != "profiles" {
		t.Fatalf("separately quoted dataset-dot completion should contain only tables/views: %#v, %v", quotedItems, err)
	}
}

func TestMergeRemovesDuplicatesKeepingFirstOccurrence(t *testing.T) {
	first := []Item{{Label: "orders", Detail: "table"}}
	second := []Item{{Label: "orders", Detail: "keyword"}, {Label: "ORDER BY", Detail: "keyword"}}
	merged := Merge(first, second)
	if len(merged) != 2 {
		t.Fatalf("expected duplicates removed: %#v", merged)
	}
	if merged[0].Detail != "table" {
		t.Fatalf("expected first occurrence to win: %#v", merged)
	}
}

func TestMergeWithoutSetsIsEmpty(t *testing.T) {
	if merged := Merge(); len(merged) != 0 {
		t.Fatalf("expected an empty merge: %#v", merged)
	}
}

func TestChainSkipsFailingProvidersAndReportsNoResults(t *testing.T) {
	chain := Chain{Providers: []Provider{
		providerFunc(func(context.Context, Request) ([]Item, error) { return nil, errors.New("unavailable") }),
		providerFunc(func(context.Context, Request) ([]Item, error) { return nil, nil }),
	}}
	items, err := chain.Complete(context.Background(), Request{})
	if err != nil || len(items) != 0 {
		t.Fatalf("expected no completions: %#v, %v", items, err)
	}
}

func TestWordPrefixClampsTheCursor(t *testing.T) {
	if got := WordPrefix("orders", 99); got != "orders" {
		t.Fatalf("cursor past the end = %q, want %q", got, "orders")
	}
	if got := WordPrefix("orders", -5); got != "" {
		t.Fatalf("negative cursor = %q, want empty", got)
	}
}

func TestReferencePartsWithoutATokenIsEmpty(t *testing.T) {
	if parts := ReferenceParts("SELECT * FROM ", 14); parts != nil {
		t.Fatalf("expected no reference parts: %#v", parts)
	}
}

func TestReferenceIsQuotedAtTheStartOfTheQuery(t *testing.T) {
	if ReferenceIsQuoted("", 0) {
		t.Fatal("an empty query has no quoted reference")
	}
	if !ReferenceIsQuoted("`orders", 7) {
		t.Fatal("a leading backtick marks a quoted reference")
	}
}

func TestProvidersReturnNothingWithoutAPrefix(t *testing.T) {
	request := Request{Project: "demo-project", SQL: "SELECT * FROM ", Cursor: 14}
	catalog := CatalogProvider{Catalog: []project.Project{{ID: "demo-project", Resources: []project.Resource{{Name: "customers", Kind: "dataset"}}}}}
	for name, provider := range map[string]Provider{"catalog": catalog, "keyword": KeywordProvider{}, "function": FunctionProvider{}} {
		items, err := provider.Complete(context.Background(), request)
		if err != nil || len(items) != 0 {
			t.Fatalf("%s provider returned suggestions without a prefix: %#v, %v", name, items, err)
		}
	}
}

func TestCatalogProviderSortsSuggestionsAlphabetically(t *testing.T) {
	provider := CatalogProvider{Catalog: []project.Project{{
		ID: "demo-project",
		Resources: []project.Resource{{Name: "customers", Kind: "dataset", Children: []project.Resource{
			{Name: "customer_zones", Kind: "table"},
			{Name: "customer_addresses", Kind: "table"},
		}}},
	}}}
	items, err := provider.Complete(context.Background(), Request{Project: "demo-project", SQL: "FROM demo-project.customers.customer", Cursor: len("FROM demo-project.customers.customer")})
	if err != nil || len(items) != 2 {
		t.Fatalf("unexpected suggestions: %#v, %v", items, err)
	}
	if items[0].Label != "customer_addresses" || items[1].Label != "customer_zones" {
		t.Fatalf("suggestions are not sorted: %#v", items)
	}
}
