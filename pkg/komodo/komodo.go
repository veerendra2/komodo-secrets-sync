package komodo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

const (
	WriteCreateVariable            = "CreateVariable"
	WriteDeleteVariable            = "DeleteVariable"
	WriteUpdateVariableDescription = "UpdateVariableDescription"
	WriteUpdateVariableIsSecret    = "UpdateVariableIsSecret"
	WriteUpdateVariableValue       = "UpdateVariableValue"
)

type Config struct {
	URL       string `name:"url" help:"Komodo URL" env:"URL" required:""`
	ApiKey    string `name:"api-key" help:"Komodo API key" env:"API_KEY" required:""`
	ApiSecret string `name:"api-secret" help:"Komodo API secret" env:"API_SECRET" required:""`
}

type Client interface {
	DeleteVariable(ctx context.Context, name string) error
	UpdateVariableDescription(ctx context.Context, name string, description string) error
	UpdateVariableIsSecret(ctx context.Context, name string, is_secret bool) error
	UpdateVariableValue(ctx context.Context, name string, value string) error
	UpsertVariable(ctx context.Context, name string, value string, description string, is_secret bool) error

	doRequest(ctx context.Context, typ string, params any) error
}

type client struct {
	httpClient http.Client

	baseUrl   *url.URL
	apiKey    string
	apiSecret string
}

func (c *client) doRequest(ctx context.Context, typ string, params any) error {
	// Support only 'write' module at the moment
	// https://docs.rs/komodo_client/latest/komodo_client/api/write/index.html
	requestURL := strings.TrimRight(c.baseUrl.String(), "/") + "/write"

	payload := Request{
		Type:   typ,
		Params: params,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("X-Api-Secret", c.apiSecret)

	slog.Debug("Sending HTTP request...", "request_url", requestURL, "payload", payload)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("komodo API error: status=%d body=%s", resp.StatusCode, respBody)
	}

	return nil
}

func (c *client) UpdateVariableIsSecret(ctx context.Context, name string, is_secret bool) error {
	params := UpdateVariableIsSecret{
		Name:     name,
		IsSecret: is_secret,
	}
	return c.doRequest(ctx, WriteUpdateVariableIsSecret, params)
}

func (c *client) UpsertVariable(ctx context.Context, name string, value string, description string, is_secret bool) error {
	params := CreateVariable{
		Name:        name,
		Value:       value,
		Description: description,
		IsSecret:    is_secret,
	}

	err := c.doRequest(ctx, WriteCreateVariable, params)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			slog.Warn("Variable already exists, updating...", "variable", name)

			err = c.UpdateVariableDescription(ctx, name, description)
			if err != nil {
				return fmt.Errorf("error while updating existing variable description: %w", err)
			}

			err = c.UpdateVariableIsSecret(ctx, name, is_secret)
			if err != nil {
				return fmt.Errorf("error while updating existing variable is_secret: %w", err)
			}

			err = c.UpdateVariableValue(ctx, name, value)
			if err != nil {
				return fmt.Errorf("error while updating existing variable value: %w", err)
			}
		} else {
			return err
		}

	}
	return nil
}

func (c *client) DeleteVariable(ctx context.Context, name string) error {
	params := DeleteVariable{
		Name: name,
	}

	return c.doRequest(ctx, WriteDeleteVariable, params)
}

func (c *client) UpdateVariableDescription(ctx context.Context, name string, description string) error {
	params := UpdateVariableDescription{
		Name:        name,
		Description: description,
	}

	return c.doRequest(ctx, WriteUpdateVariableDescription, params)
}

func (c *client) UpdateVariableValue(ctx context.Context, name string, value string) error {
	params := UpdateVariableValue{
		Name:  name,
		Value: value,
	}

	return c.doRequest(ctx, WriteUpdateVariableValue, params)
}

func NewClient(cfg Config) (Client, error) {
	baseUrl, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}

	return &client{
		httpClient: http.Client{},

		baseUrl:   baseUrl,
		apiKey:    cfg.ApiKey,
		apiSecret: cfg.ApiSecret,
	}, nil
}
