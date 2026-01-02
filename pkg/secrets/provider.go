package secrets

import (
	"context"
)

type Client interface {
	Get(ctx context.Context, id string) (string, error)
	Dump(ctx context.Context) (*Dump, error)
}
