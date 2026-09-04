package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kubometr/internal/ai"
	"kubometr/internal/config"
	"kubometr/internal/consultation"
	"kubometr/internal/database"
	"kubometr/internal/history"
	"kubometr/internal/logger"
	"kubometr/internal/max"
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

	pool, err := database.Connect(ctx, &cfg)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	s := state.New()

	a, err := ai.New(cfg.AIAPIKey, cfg.AIModel)
	if err != nil {
		return fmt.Errorf("create ai client: %w", err)
	}

	h := history.New(pool)

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
		ProxyURL: cfg.ProxyURL,
	})
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", err)
	}

	if cfg.MaxToken != "" {
		maxHandler := max.NewHandler(cs, cfg.MaxWebhookSecret)
		mux := http.NewServeMux()
		mux.HandleFunc("/max/webhook", maxHandler.HandleWebhook)
		srv := &http.Server{
			Addr: fmt.Sprintf(":%d", cfg.MaxPort),
			Handler: mux,
			ReadHeaderTimeout: 5 * time.Second,
		}
		go func(){
			srv.ListenAndServe()
		}()
	}
	
	tg.Start(ctx)
	return nil
}