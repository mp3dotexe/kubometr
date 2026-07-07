package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken        string
	GeminiAPIKey    string
	GeminiModel     string
	AITimeout       time.Duration
	AIRateLimit     time.Duration
	MaxPromptLength int
	MaxConcurrentAI int
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	token := strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	geminiAPIKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if geminiAPIKey == "" {
		return Config{}, errors.New("GEMINI_API_KEY is required")
	}
	if token == "" {
		return Config{}, errors.New("BOT_TOKEN is required")
	}

	aiTimeout, err := durationFromEnv("AI_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	aiRateLimit, err := durationFromEnv("AI_RATE_LIMIT", 3*time.Second)
	if err != nil {
		return Config{}, err
	}

	maxPromptLength, err := intFromEnv("MAX_PROMPT_LENGTH", 2000)
	if err != nil {
		return Config{}, err
	}

	maxConcurrentAI, err := intFromEnv("MAX_CONCURRENT_AI", 4)
	if err != nil {
		return Config{}, err
	}

	return Config{
		BotToken:        token,
		GeminiAPIKey:    geminiAPIKey,
		GeminiModel:     stringFromEnv("GEMINI_MODEL", "gemini-2.5-flash"),
		AITimeout:       aiTimeout,
		AIRateLimit:     aiRateLimit,
		MaxPromptLength: maxPromptLength,
		MaxConcurrentAI: maxConcurrentAI,
	}, nil
}

func stringFromEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", key, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}

	return duration, nil
}

func intFromEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	if number <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}

	return number, nil
}
