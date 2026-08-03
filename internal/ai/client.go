package ai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	client openai.Client
	model  string
}

func New(apiKey, model string) (*Client, error) {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://openrouter.ai/api/v1"),
	)

	return &Client{
		client: client,
		model:  model,
	}, nil
}

func (c *Client) Ask(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model: c.model,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("chat completion: %w", err)
	}

	return resp.Choices[0].Message.Content, nil
}
