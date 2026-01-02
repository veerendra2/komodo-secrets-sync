package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bitwarden/sdk-go"
)

type BitwardenConfig struct {
	ApiURL      string `name:"api-url" help:"API URL" env:"BW_API_URL" default:"vault.bitwarden.com/api"`
	IdentityURL string `name:"identity-url" help:"Identity URL" env:"BW_IDENTITY_URL" default:"vault.bitwarden.com/identity"`
	AccessToken string `name:"access-token" help:"Access token" env:"BW_ACCESS_TOKEN" required:""`
	OrgId       string `name:"organization-id" help:"Organization ID" env:"BW_ORGANIZATION_ID" required:""`
	ProjectName string `name:"project-name" help:"Project name" env:"BW_PROJECT_ID" required:""`
}

type bitwardenClient struct {
	organizationId string
	projectName    string

	bwClient sdk.BitwardenClientInterface
}

func (c *bitwardenClient) Dump(ctx context.Context) (*Dump, error) {
	type result struct {
		resp any
		err  error
	}

	resultChan := make(chan result, 1)

	// Bitwarden SDK doesn't support context natively, so we run the SDK call in a goroutine
	// and use select to handle context cancellation/timeout while waiting for the result
	go func() {
		resp, err := c.bwClient.Secrets().Sync(c.organizationId, nil)
		resultChan <- result{resp, err}
	}()

	var resp any
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultChan:
		if res.err != nil {
			return nil, res.err
		}
		resp = res.resp
	}

	// Marshal and unmarshal to convert the response
	jsonData, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sync response: %w", err)
	}

	var bwResp BitwardenSyncResponse
	if err := json.Unmarshal(jsonData, &bwResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sync response: %w", err)
	}

	// Convert to our Secret format
	var secretsDump Dump
	for _, bwSecret := range bwResp.Secrets {
		secretsDump.Secrets = append(secretsDump.Secrets, Secret{
			Key:   bwSecret.Key,
			Note:  bwSecret.Note,
			Value: bwSecret.Value,
		})
	}

	return &secretsDump, nil
}

func (c *bitwardenClient) Get(ctx context.Context, secret_id string) (string, error) {
	type result struct {
		value string
		err   error
	}

	resultChan := make(chan result, 1)

	// Bitwarden SDK doesn't support context natively, so we run the SDK call in a goroutine
	// and use select to handle context cancellation/timeout while waiting for the result
	go func() {
		secret, err := c.bwClient.Secrets().Get(secret_id)
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
		projectName:    cfg.ProjectName,
		bwClient:       bwClient,
	}, nil
}
