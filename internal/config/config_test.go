package config

import (
	"strings"
	"testing"
	"time"
)

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("BOT_TOKEN", "bot-token")
	t.Setenv("MAX_TOKEN", "")
	t.Setenv("MAX_WEBHOOK_SECRET", "")
	t.Setenv("OPENROUTER_API_KEY", "api-key")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "kubometr_db")
}

func TestLoadDefaults(t *testing.T) {
	setRequired(t)
	t.Setenv("BOT_TOKEN", " bot-token ")
	t.Setenv("OPENROUTER_API_KEY", " api-key ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.BotToken != "bot-token" {
		t.Fatalf("BotToken = %q", cfg.BotToken)
	}
	if cfg.AIAPIKey != "api-key" {
		t.Fatalf("AIAPIKey = %q", cfg.AIAPIKey)
	}
	if cfg.AIBaseURL != "https://openrouter.ai/api/v1" {
		t.Fatalf("AIBaseURL = %q", cfg.AIBaseURL)
	}
	if cfg.AIModel != "openai/gpt-oss-20b:free" {
		t.Fatalf("AIModel = %q", cfg.AIModel)
	}
	if cfg.AITimeout != 30*time.Second {
		t.Fatalf("AITimeout = %v", cfg.AITimeout)
	}
	if cfg.AIRateLimit != 3*time.Second {
		t.Fatalf("AIRateLimit = %v", cfg.AIRateLimit)
	}
	if cfg.MaxPromptLength != 2000 {
		t.Fatalf("MaxPromptLength = %d", cfg.MaxPromptLength)
	}
	if cfg.MaxConcurrentAI != 4 {
		t.Fatalf("MaxConcurrentAI = %d", cfg.MaxConcurrentAI)
	}
	if cfg.MaxPort != 8080 {
		t.Fatalf("MaxPort = %d", cfg.MaxPort)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	setRequired(t)
	t.Setenv("AI_TIMEOUT", "soon")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "AI_TIMEOUT") {
		t.Fatalf("Load() error = %v, want AI_TIMEOUT error", err)
	}
}

func TestLoadRequiresMessenger(t *testing.T) {
	setRequired(t)
	t.Setenv("BOT_TOKEN", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error without any messenger token")
	}
}

func TestLoadMaxRequiresSecret(t *testing.T) {
	setRequired(t)
	t.Setenv("MAX_TOKEN", "max-token")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "MAX_WEBHOOK_SECRET") {
		t.Fatalf("Load() error = %v, want MAX_WEBHOOK_SECRET error", err)
	}
}

func TestLoadMaxOnly(t *testing.T) {
	setRequired(t)
	t.Setenv("BOT_TOKEN", "")
	t.Setenv("MAX_TOKEN", "max-token")
	t.Setenv("MAX_WEBHOOK_SECRET", "secret")
	t.Setenv("MAX_WEBHOOK_URL", "https://example.com/max/webhook")
	t.Setenv("MAX_PORT", "9090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.BotToken != "" || cfg.MaxToken != "max-token" {
		t.Fatalf("tokens = %q, %q", cfg.BotToken, cfg.MaxToken)
	}
	if cfg.MaxWebhookURL != "https://example.com/max/webhook" || cfg.MaxPort != 9090 {
		t.Fatalf("webhook = %q, port = %d", cfg.MaxWebhookURL, cfg.MaxPort)
	}
}
