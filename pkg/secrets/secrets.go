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

// Client defines the interface for interacting with secrets managers
type Client interface {
	// Get retrieves a single secret by ID
	Get(ctx context.Context, id string) (string, error)

	// FetchAll retrieves all secrets from the secrets manager
	FetchAll(ctx context.Context) (*SecretsCollection, error)

	// Close cleans up resources and closes connections
	Close() error
}
