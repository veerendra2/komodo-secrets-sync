package secrets

import (
	"context"
)

type Secret struct {
	Key   string
	Note  string
	Value string
}

type SecretsCollection struct {
	Secrets []Secret
}

type Client interface {
	Get(ctx context.Context, id string) (string, error)
	FetchAll(ctx context.Context) (*SecretsCollection, error)
}
