package secrets

import (
	"context"
	"fmt"
	"strings"

	"github.com/bitwarden/sdk-go"
)

type BitwardenConfig struct {
	ApiURL      string `name:"api-url" help:"API URL" env:"BW_API_URL" default:"vault.bitwarden.com/api"`
	IdentityURL string `name:"identity-url" help:"Identity URL" env:"BW_IDENTITY_URL" default:"vault.bitwarden.com/identity"`
	AccessToken string `name:"access-token" help:"Access token" env:"BW_ACCESS_TOKEN" required:""`
	OrgId       string `name:"organization-id" help:"Organization ID" env:"BW_ORGANIZATION_ID" required:""`
	ProjectId   string `name:"project-id" help:"Project ID" env:"BW_PROJECT_ID" required:""`
}

type bitwardenClient struct {
	organizationId string
	projectId      string

	bwClient sdk.BitwardenClientInterface
}

// FetchAll retrieves all secrets from Bitwarden.
// Note: The Bitwarden SDK doesn't support context cancellation, so the ctx parameter
// is accepted for interface compatibility but not used. SDK calls will run to completion.
func (c *bitwardenClient) FetchAll(_ context.Context) (*SecretsCollection, error) {
	resp, err := c.bwClient.Secrets().Sync(c.organizationId, nil)
	if err != nil {
		return nil, err
	}

	// Convert to our Secret format
	var secretsCollection SecretsCollection
	for _, bwSecret := range resp.Secrets {
		// Filter secrets by project ID
		// Note: Sync() already returns only secrets the access token has permission to access,
		// but we filter by project ID to ensure we only sync secrets from the specified project
		if bwSecret.ProjectID != nil && *bwSecret.ProjectID == c.projectId {
			secretsCollection.Secrets = append(secretsCollection.Secrets, Secret{
				Key:   bwSecret.Key,
				Note:  bwSecret.Note,
				Value: bwSecret.Value,
			})
		}
	}

	return &secretsCollection, nil
}

// Get retrieves a single secret by ID.
// Note: The Bitwarden SDK doesn't support context cancellation, so the ctx parameter
// is accepted for interface compatibility but not used. SDK calls will run to completion.
func (c *bitwardenClient) Get(_ context.Context, id string) (string, error) {
	secret, err := c.bwClient.Secrets().Get(id)
	if err != nil {
		return "", err
	}
	return secret.Value, nil
}

// Close cleans up the Bitwarden client resources
func (c *bitwardenClient) Close() error {
	if c.bwClient != nil {
		c.bwClient.Close()
	}
	return nil
}

func NewBitwarden(cfg BitwardenConfig) (Client, error) {
	apiEndpoint := cfg.ApiURL
	identityEndpoint := cfg.IdentityURL

	if !strings.HasPrefix(apiEndpoint, "http") {
		apiEndpoint = fmt.Sprintf("https://%s", apiEndpoint)
	}
	if !strings.HasPrefix(identityEndpoint, "http") {
		identityEndpoint = fmt.Sprintf("https://%s", identityEndpoint)
	}

	bwClient, err := sdk.NewBitwardenClient(&apiEndpoint, &identityEndpoint)
	if err != nil {
		return nil, err
	}

	err = bwClient.AccessTokenLogin(cfg.AccessToken, nil)
	if err != nil {
		return nil, err
	}

	return &bitwardenClient{
		organizationId: cfg.OrgId,
		projectId:      cfg.ProjectId,
		bwClient:       bwClient,
	}, nil
}
