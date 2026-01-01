package secretsmanager

import (
	"fmt"
	"strings"

	"github.com/bitwarden/sdk-go"
)

type BitwardenConfig struct {
	ApiURL      string `name:"api-url" help:"API URL" env:"API_URL" default:"api.bitwarden.com"`
	IdentityURL string `name:"identity-url" help:"Identity URL" env:"IDENTITY_URL" default:"identity.bitwarden.com/connect/token"`
	AccessToken string `name:"access-token" help:"Access token" env:"ACCESS_TOKEN" required:""`
	OrgId       string `name:"organization-id" help:"Organization ID" env:"ORGANIZATION_ID" required:""`
	ProjectName string `name:"project-name" help:"Project name" env:"PROJECT_ID" required:""`
}

type bitwardenClient struct {
	organizationId string
	projectName    string

	bwClient sdk.BitwardenClientInterface
}

func (c *bitwardenClient) Get(secret_id string) (string, error) {
	secret, err := c.bwClient.Secrets().Get(secret_id)
	if err != nil {
		return "", err
	}

	return secret.Value, nil
}

func NewBwClient(cfg BitwardenConfig) (Client, error) {
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
		fmt.Println("here!")
		return nil, err
	}

	return &bitwardenClient{
		organizationId: cfg.OrgId,
		projectName:    cfg.ProjectName,
		bwClient:       bwClient,
	}, nil
}
