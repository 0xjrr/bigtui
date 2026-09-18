package completion

import (
	"context"
	"testing"
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
