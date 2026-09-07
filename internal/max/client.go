package max

import (
	"fmt"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
)

type Client struct{
	api 	*maxbot.Api
}

func NewClient(token string) (*Client, error) {
	mx, err := maxbot.New(token)
	if err != nil {
		return nil, fmt.Errorf("create max client: %w", err)
	}
	return &Client{api: mx}, nil
}