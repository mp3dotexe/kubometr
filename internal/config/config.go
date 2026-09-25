package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/joho/godotenv"
)

// defaultAIModels are free OpenRouter models tried in order. Free models get
// rate-limited or withdrawn without notice, hence several of them.
var defaultAIModels = []string{
	"nvidia/nemotron-3-super-120b-a12b:free",
	"qwen/qwen3.8-27b:free",
	"google/gemma-4-31b-it:free",
}

type Config struct {
	BotToken         string
	ProxyURL         string
	MaxToken         string
	MaxWebhookSecret string
	MaxWebhookURL    string
	MaxPort          int
	AIAPIKey         string
	AIBaseURL        string
	AIModels         []string
	AITimeout        time.Duration
	AIModelTimeout   time.Duration
	AIRateLimit      time.Duration
	MaxPromptLength  int
	MaxConcurrentAI  int
	ManagerChatID    int64
	ManagerContact   string

	PostgresHost     string
	PostgresPort     int
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	token := strings.TrimSpace(os.Getenv("BOT_TOKEN"))

	maxToken := strings.TrimSpace(os.Getenv("MAX_TOKEN"))
	maxWebhookSecret := strings.TrimSpace(os.Getenv("MAX_WEBHOOK_SECRET"))
	if maxToken != "" && maxWebhookSecret == "" {
		return Config{}, errors.New("MAX_WEBHOOK_SECRET is required when MAX_TOKEN is set")
	}
	if token == "" && maxToken == "" {
		return Config{}, errors.New("at least one of BOT_TOKEN or MAX_TOKEN is required")
	}
	maxPort, err := intFromEnv("MAX_PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	proxyURL := strings.TrimSpace(os.Getenv("PROXY_URL"))

	// Group chat IDs in MAX are negative, so any non-zero number is valid.
	var managerChatID int64
	if value := strings.TrimSpace(os.Getenv("MANAGER_CHAT_ID")); value != "" {
		managerChatID, err = strconv.ParseInt(value, 10, 64)
		if err != nil || managerChatID == 0 {
			return Config{}, errors.New("MANAGER_CHAT_ID must be a non-zero MAX chat ID")
		}
		if maxToken == "" {
			return Config{}, errors.New("MANAGER_CHAT_ID is a MAX chat and requires MAX_TOKEN")
		}
	}

	aiAPIKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if aiAPIKey == "" {
		return Config{}, errors.New("OPENROUTER_API_KEY is required")
	}

	aiTimeout, err := durationFromEnv("AI_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	aiModelTimeout, err := durationFromEnv("AI_MODEL_TIMEOUT", 20*time.Second)
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

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}

	port, err := intFromEnv("POSTGRES_PORT", 5432)
	if err != nil {
		return Config{}, err
	}

	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		return Config{}, errors.New("POSTGRES_USER is required")
	}

	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		return Config{}, errors.New("POSTGRES_PASSWORD is required")
	}

	database := os.Getenv("POSTGRES_DB")
	if database == "" {
		return Config{}, errors.New("POSTGRES_DB is required")
	}

	return Config{
		BotToken:         token,
		ProxyURL:         proxyURL,
		MaxToken:         maxToken,
		MaxWebhookSecret: maxWebhookSecret,
		MaxWebhookURL:    strings.TrimSpace(os.Getenv("MAX_WEBHOOK_URL")),
		MaxPort:          maxPort,
		AIAPIKey:         aiAPIKey,
		AIBaseURL:        stringFromEnv("AI_BASE_URL", "https://openrouter.ai/api/v1"),
		AIModels:         listFromEnv("AI_MODEL", defaultAIModels),
		AITimeout:        aiTimeout,
		AIModelTimeout:   aiModelTimeout,
		AIRateLimit:      aiRateLimit,
		MaxPromptLength:  maxPromptLength,
		MaxConcurrentAI:  maxConcurrentAI,
		ManagerChatID:    managerChatID,
		ManagerContact:   strings.TrimSpace(os.Getenv("MANAGER_CONTACT")),
		PostgresHost:     host,
		PostgresPort:     port,
		PostgresUser:     user,
		PostgresPassword: password,
		PostgresDB:       database,
	}, nil
}

func listFromEnv(key string, fallback []string) []string {
	values := strings.FieldsFunc(os.Getenv(key), func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
	if len(values) == 0 {
		return fallback
	}
	return values
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
