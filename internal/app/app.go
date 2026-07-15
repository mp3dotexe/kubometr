package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"kubometr/internal/ai"
	"kubometr/internal/config"
	"kubometr/internal/consultation"
	"kubometr/internal/database"
	"kubometr/internal/history"
	"kubometr/internal/logger"
	"kubometr/internal/state"
	"kubometr/internal/telegram"
	"kubometr/internal/users"
)

func Run() error {
	slog.SetDefault(logger.New())

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.New(ctx, &cfg)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	s := state.New()

	a, err := ai.New(cfg.GeminiAPIKey, cfg.GeminiModel)
	if err != nil {
		return fmt.Errorf("create ai client: %w", err)
	}

	h := history.New()

	usersRepo := users.New(pool)

	cs := consultation.New(
		s,
		a,
		cfg.AITimeout,
		cfg.AIRateLimit,
		cfg.MaxPromptLength,
		cfg.MaxConcurrentAI,
		h,
		usersRepo,
	)

	tg, err := telegram.New(telegram.Options{
		Token:        cfg.BotToken,
		Consultation: cs,
	})
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", err)
	}

	tg.Start(ctx)
	return nil
}
