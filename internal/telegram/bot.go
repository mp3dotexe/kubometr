package telegram

import (
	"context"
	"kubometr/internal/consultation"
	"net/http"
	"net/url"
	"time"

	"github.com/go-telegram/bot"
	"golang.org/x/net/proxy"
)

type Options struct {
	Token        string
	Consultation *consultation.Service
	ProxyURL	string
}

type Telegram struct {
	bot          *bot.Bot
	consultation *consultation.Service
}

func New(opts Options) (*Telegram, error) {
	var opts_bot []bot.Option
	if opts.ProxyURL != "" {
		proxyURL, err := url.Parse(opts.ProxyURL)
		if err != nil {
			return nil, err
		}

		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, err
		}

		httpClient := &http.Client{
			Transport: &http.Transport{
				DialContext: dialer.(proxy.ContextDialer).DialContext,
			},
			Timeout: 30 * time.Second,
		}

		opts_bot = append(opts_bot, bot.WithHTTPClient(10*time.Second, httpClient))
	}
	b, err := bot.New(opts.Token, opts_bot...)
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
