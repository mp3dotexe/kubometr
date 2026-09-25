package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"kubometr/internal/chat"
	"kubometr/internal/consultation"
	"kubometr/internal/requests"

	"github.com/go-telegram/bot"
	"golang.org/x/net/proxy"
)

type Options struct {
	Token          string
	Consultation   *consultation.Service
	Requests       *requests.Service
	ManagerContact string
	ProxyURL       string
}

type Telegram struct {
	bot            *bot.Bot
	consultation   *consultation.Service
	requests       *requests.Service
	managerContact string
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
		bot:            b,
		consultation:   opts.Consultation,
		requests:       opts.Requests,
		managerContact: opts.ManagerContact,
	}
	t.registerHandlers()

	return t, nil
}

func (t *Telegram) Start(ctx context.Context) {
	// The commands show up in the "Menu" button next to the input field.
	if _, err := t.bot.SetMyCommands(ctx, &bot.SetMyCommandsParams{Commands: commands}); err != nil {
		slog.ErrorContext(ctx, "set telegram commands", "error", err)
	}
	t.bot.Start(ctx)
}

// registerHandlers registers handlers in order: the first matching one wins,
// so the catch-all consultation handler goes last.
func (t *Telegram) registerHandlers() {
	exact := func(text string, handler bot.HandlerFunc) {
		t.bot.RegisterHandler(bot.HandlerTypeMessageText, text, bot.MatchTypeExact, handler)
	}
	exact("/start", t.HandleStart)
	exact("/help", t.HandleHelp)
	exact(chat.ButtonHelp, t.HandleHelp)
	exact(chat.ButtonConsultation, t.HandleConsultation)
	exact("/new", t.HandleNewDialog)
	exact(chat.ButtonNewDialog, t.HandleNewDialog)
	exact("/requests", t.HandleRequests)
	exact(chat.ButtonRequests, t.HandleRequests)
	exact("/manager", t.HandleManager)
	exact(chat.ButtonManager, t.HandleManager)
	// A contact message has no text and would match the catch-all below.
	t.bot.RegisterHandlerMatchFunc(hasContact, t.HandleContact)
	t.bot.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypePrefix, t.HandleMessage)
}
