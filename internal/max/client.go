package max

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

type Client struct {
	api *maxbot.Api
}

func NewClient(token string, httpClient *http.Client) (*Client, error) {
	mx, err := maxbot.New(token, maxbot.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create max client: %w", err)
	}
	return &Client{api: mx}, nil
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	msg := maxbot.NewMessage().SetChat(chatID).SetText(text)
	err := c.api.Messages.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

func (c *Client) SendTyping(ctx context.Context, chatID int64) error {
	result, err := c.api.Chats.SendAction(ctx, chatID, schemes.TYPING_ON)
	if err != nil {
		return fmt.Errorf("send typing: %w", err)
	}
	if !result.Success {
		return errors.New("send typing: " + result.Message)
	}
	return nil
}

// Subscribe registers webhookURL so MAX starts delivering updates to it.
// Registering the same URL again just updates the subscription.
func (c *Client) Subscribe(ctx context.Context, webhookURL, secret string) error {
	result, err := c.api.Subscriptions.Subscribe(
		ctx,
		webhookURL,
		[]string{updateMessageCreated, updateBotStarted},
		secret,
	)
	if err != nil {
		return fmt.Errorf("subscribe webhook: %w", err)
	}
	if !result.Success {
		return errors.New("subscribe webhook: " + result.Message)
	}
	return nil
}
