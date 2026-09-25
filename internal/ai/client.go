package ai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role
	Content string
}

type Client struct {
	client openai.Client
	model  string
}

// New creates a client for any OpenAI-compatible chat completions API
// (OpenRouter by default).
func New(apiKey, baseURL, model string) (*Client, error) {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)

	return &Client{
		client: client,
		model:  model,
	}, nil
}

func (c *Client) Complete(ctx context.Context, messages []Message) (string, error) {
	params := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case RoleSystem:
			params = append(params, openai.SystemMessage(m.Content))
		case RoleUser:
			params = append(params, openai.UserMessage(m.Content))
		case RoleAssistant:
			params = append(params, openai.AssistantMessage(m.Content))
		default:
			return "", fmt.Errorf("unknown message role %q", m.Role)
		}
	}

	resp, err := c.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model:    c.model,
			Messages: params,
		},
	)
	if err != nil {
		return "", fmt.Errorf("chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from chat completion")
	}

	return resp.Choices[0].Message.Content, nil
}
