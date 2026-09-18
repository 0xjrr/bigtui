package completion

import "context"

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
