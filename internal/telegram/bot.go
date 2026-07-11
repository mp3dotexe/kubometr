package telegram

import (
	"context"
	"kubometr/internal/consultation"

	"github.com/go-telegram/bot"
)

type Options struct {
	Token        string
	Consultation *consultation.Service
}

type Telegram struct {
	bot          *bot.Bot
	consultation *consultation.Service
}

func New(opts Options) (*Telegram, error) {
	b, err := bot.New(opts.Token)
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
