package ai

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type Client struct {
	client *genai.Client
	model  string
}

func New(apiKey, model string) (*Client, error) {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	return &Client{
		client: client,
		model:  model,
	}, nil
}

func (c *Client) Ask(ctx context.Context, prompt string) (string, error) {
	
	resp, err := c.client.Models.GenerateContent(
		ctx,
		c.model,
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("generate content: %w", err)
	}

	return resp.Text(), nil
}
