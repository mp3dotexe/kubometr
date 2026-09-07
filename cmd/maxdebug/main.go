package main

import(
	"context"
	"os"
	"fmt"
	"kubometr/internal/max"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
)

func main() {
	httpClient, err := max.NewTrustedHTTPClient()
	if err != nil {
		panic(err)
	}
	
	api, err := maxbot.New(os.Getenv("MAX_TOKEN"), maxbot.WithHTTPClient(httpClient))
	if err != nil {
		panic(err)
	}

	updates := api.GetUpdates(context.Background())
	for u := range updates {
		fmt.Printf("%+v\n", u)
	}
}
