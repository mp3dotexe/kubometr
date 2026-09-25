package telegram

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"kubometr/internal/consultation"

	"github.com/go-telegram/bot"
	"golang.org/x/net/proxy"
)

type Options struct {
	Token        string
	Consultation *consultation.Service
	ProxyURL     string
}

type Telegram struct {
	bot          *bot.Bot
	consultation *consultation.Service
}

func New(opts Options) (*Telegram, error) {
	var botOpts []bot.Option
	if opts.ProxyURL != "" {
		proxyURL, err := url.Parse(opts.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("parse proxy url: %w", err)
		}

		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("create proxy dialer: %w", err)
		}

		contextDialer, ok := dialer.(proxy.ContextDialer)
		if !ok {
			return nil, fmt.Errorf("proxy scheme %q does not support dialing with context", proxyURL.Scheme)
		}

		httpClient := &http.Client{
			Transport: &http.Transport{
				DialContext: contextDialer.DialContext,
			},
			Timeout: 30 * time.Second,
		}

		botOpts = append(botOpts, bot.WithHTTPClient(10*time.Second, httpClient))
	}
	b, err := bot.New(opts.Token, botOpts...)
	if err != nil {
		return nil, err
	}

	t := &Telegram{
		bot:          b,
		consultation: opts.Consultation,
	}
	t.registerHandlers()

	return t, nil
}

func (t *Telegram) Start(ctx context.Context) {
	t.bot.Start(ctx)
}

func (t *Telegram) registerHandlers() {
	t.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, t.HandleStart)
	t.bot.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, t.HandleHelp)
	t.bot.RegisterHandler(bot.HandlerTypeMessageText, "ℹ️ Помощь", bot.MatchTypeExact, t.HandleHelp)
	t.bot.RegisterHandler(bot.HandlerTypeMessageText, "💬 Консультация", bot.MatchTypeExact, t.HandleConsultation)
	t.bot.RegisterHandler(bot.HandlerTypeMessageText, "🧾 Мои заявки", bot.MatchTypeExact, t.HandleRequests)
	t.bot.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypePrefix, t.HandleMessage)
}
