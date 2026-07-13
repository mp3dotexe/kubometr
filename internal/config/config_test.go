package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("BOT_TOKEN", " bot-token ")
	t.Setenv("GEMINI_API_KEY", " gemini-key ")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "kubometr_db")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.BotToken != "bot-token" {
		t.Fatalf("BotToken = %q", cfg.BotToken)
	}
	if cfg.GeminiAPIKey != "gemini-key" {
		t.Fatalf("GeminiAPIKey = %q", cfg.GeminiAPIKey)
	}
	if cfg.GeminiModel != "gemini-2.5-flash" {
		t.Fatalf("GeminiModel = %q", cfg.GeminiModel)
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
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv("BOT_TOKEN", "bot-token")
	t.Setenv("GEMINI_API_KEY", "gemini-key")
	t.Setenv("AI_TIMEOUT", "soon")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "kubometr_db")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
