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
	"kubometr/internal/logger"
	"kubometr/internal/state"
	"kubometr/internal/telegram"
)

func Run() error {
	slog.SetDefault(logger.New())

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	s := state.New()

	a, err := ai.New(cfg.GeminiAPIKey, cfg.GeminiModel)
	if err != nil {
		return err
	}

	cs := consultation.New(
		s, 
		a, 
		cfg.AITimeout, 
		cfg.AIRateLimit, 
		cfg.MaxPromptLength, 
		cfg.MaxConcurrentAI,
	)
	
	tg, err := telegram.New(telegram.Options{
		Token:           cfg.BotToken,
		Consultation: cs,
	})
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tg.Start(ctx)
	return nil
}
