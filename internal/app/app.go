package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
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
	"kubometr/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

const shutdownTimeout = 10 * time.Second

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

	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	a, err := ai.New(cfg.AIAPIKey, cfg.AIBaseURL, cfg.AIModel)
	if err != nil {
		return fmt.Errorf("create ai client: %w", err)
	}

	cs := consultation.New(
		state.New(),
		a,
		cfg.AITimeout,
		cfg.AIRateLimit,
		cfg.MaxPromptLength,
		cfg.MaxConcurrentAI,
		history.New(pool),
		users.New(pool),
	)

	g, gctx := errgroup.WithContext(ctx)

	if cfg.BotToken != "" {
		tg, err := telegram.New(telegram.Options{
			Token:        cfg.BotToken,
			Consultation: cs,
			ProxyURL:     cfg.ProxyURL,
		})
		if err != nil {
			return fmt.Errorf("create telegram bot: %w", err)
		}

		g.Go(func() error {
			slog.Info("telegram bot started")
			tg.Start(gctx)
			return nil
		})
	}

	if cfg.MaxToken != "" {
		if err := startMax(gctx, g, &cfg, cs, pool); err != nil {
			return err
		}
	}

	return g.Wait()
}

func startMax(ctx context.Context, g *errgroup.Group, cfg *config.Config, cs *consultation.Service, pool *pgxpool.Pool) error {
	httpClient, err := max.NewTrustedHTTPClient()
	if err != nil {
		return fmt.Errorf("create max http client: %w", err)
	}

	client, err := max.NewClient(cfg.MaxToken, httpClient)
	if err != nil {
		return err
	}

	// Leave room for the database calls around the AI request.
	handler := max.NewHandler(cs, client, cfg.MaxWebhookSecret, cfg.AITimeout+30*time.Second)

	mux := http.NewServeMux()
	mux.HandleFunc("/max/webhook", handler.HandleWebhook)
	mux.HandleFunc("/healthz", healthz(pool))

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Bind the port before subscribing, so MAX never calls a closed port.
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.MaxPort))
	if err != nil {
		return fmt.Errorf("listen max webhook port: %w", err)
	}

	if cfg.MaxWebhookURL != "" {
		if err := client.Subscribe(ctx, cfg.MaxWebhookURL, cfg.MaxWebhookSecret); err != nil {
			ln.Close()
			return fmt.Errorf("subscribe max webhook: %w", err)
		}
		slog.Info("max webhook subscribed", "url", cfg.MaxWebhookURL)
	}

	g.Go(func() error {
		slog.Info("max webhook server started", "port", cfg.MaxPort)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve max webhook: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown max webhook server", "error", err)
		}

		// Let already accepted updates finish, so users get their answers.
		handler.Wait()
		return nil
	})

	return nil
}

func healthz(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	}
}
