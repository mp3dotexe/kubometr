package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

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
	client       openai.Client
	models       []string
	modelTimeout time.Duration
}

// New creates a client for any OpenAI-compatible chat completions API
// (OpenRouter by default). Models are tried in order: when one fails, the
// request falls back to the next, which keeps the bot answering when a free
// model is rate-limited or withdrawn. Every model except the last gets at
// most modelTimeout, so one stuck model can't eat the whole request timeout.
func New(apiKey, baseURL string, models []string, modelTimeout time.Duration) (*Client, error) {
	if len(models) == 0 {
		return nil, errors.New("at least one model is required")
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)

	return &Client{
		client:       client,
		models:       models,
		modelTimeout: modelTimeout,
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

	var errs []error
	for i, model := range c.models {
		// Retrying a failing model or waiting for a stuck one only delays the
		// fallback, so only the last model gets the SDK's default retries and
		// the rest of the request timeout.
		attemptCtx, cancel := ctx, func() {}
		var opts []option.RequestOption
		if i < len(c.models)-1 {
			opts = append(opts, option.WithMaxRetries(0))
			attemptCtx, cancel = context.WithTimeout(ctx, c.modelTimeout)
		}

		answer, err := c.complete(attemptCtx, model, params, opts...)
		cancel()
		if err == nil {
			return answer, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", model, err))

		if ctx.Err() != nil || isUnauthorized(err) || i == len(c.models)-1 {
			break
		}
		slog.WarnContext(ctx, "ai model failed, falling back", "model", model, "next", c.models[i+1], "error", err)
	}

	return "", fmt.Errorf("chat completion: %w", errors.Join(errs...))
}

func (c *Client) complete(
	ctx context.Context,
	model string,
	messages []openai.ChatCompletionMessageParamUnion,
	opts ...option.RequestOption,
) (string, error) {
	start := time.Now()
	resp, err := c.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model:    model,
			Messages: messages,
		},
		opts...,
	)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("no choices returned")
	}

	answer := resp.Choices[0].Message.Content
	if strings.TrimSpace(answer) == "" {
		return "", errors.New("empty answer")
	}

	slog.InfoContext(ctx, "ai answered",
		"model", model,
		"duration", time.Since(start).Round(100*time.Millisecond),
		"completion_tokens", resp.Usage.CompletionTokens,
		"reasoning_tokens", resp.Usage.CompletionTokensDetails.ReasoningTokens,
	)

	return answer, nil
}

// isUnauthorized reports an invalid API key: every model shares it, so
// falling back would fail the same way.
func isUnauthorized(err error) bool {
	var apiErr *openai.Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}
