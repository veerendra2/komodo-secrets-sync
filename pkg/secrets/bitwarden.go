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
	ProjectId      string

	bwClient sdk.BitwardenClientInterface
}

func (c *bitwardenClient) FetchAll(ctx context.Context) (*SecretsCollection, error) {
	type result struct {
		resp *sdk.SecretsSyncResponse
		err  error
	}

	resultChan := make(chan result, 1)

	// Bitwarden SDK doesn't support context natively, so we run the SDK call in a goroutine
	// and use select to handle context cancellation/timeout while waiting for the result
	go func() {
		resp, err := c.bwClient.Secrets().Sync(c.organizationId, nil)
		resultChan <- result{resp, err}
	}()

	var syncResp *sdk.SecretsSyncResponse
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultChan:
		if res.err != nil {
			return nil, res.err
		}
		syncResp = res.resp
	}

	// Convert to our Secret format
	var secretsCollection SecretsCollection
	for _, bwSecret := range syncResp.Secrets {
		// Filter secrets to include only those from the specified project.
		// The SDK already limits results based on access token permissions.
		if bwSecret.ProjectID != nil && *bwSecret.ProjectID == c.ProjectId {
			secretsCollection.Secrets = append(secretsCollection.Secrets, Secret{
				Key:   bwSecret.Key,
				Note:  bwSecret.Note,
				Value: bwSecret.Value,
			})
		}
	}

	return &secretsCollection, nil
}

func (c *bitwardenClient) Get(ctx context.Context, id string) (string, error) {
	type result struct {
		value string
		err   error
	}

	resultChan := make(chan result, 1)

	// Bitwarden SDK doesn't support context natively, so we run the SDK call in a goroutine
	// and use select to handle context cancellation/timeout while waiting for the result
	go func() {
		secret, err := c.bwClient.Secrets().Get(id)
		if err != nil {
			resultChan <- result{"", err}
			return
		}
		resultChan <- result{secret.Value, nil}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-resultChan:
		return res.value, res.err
	}
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
		ProjectId:      cfg.ProjectId,
		bwClient:       bwClient,
	}, nil
}
