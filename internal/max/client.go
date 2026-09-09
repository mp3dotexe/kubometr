package max

import (
	"context"
	"fmt"
	"net/http"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
)

type Client struct{
	api 	*maxbot.Api
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