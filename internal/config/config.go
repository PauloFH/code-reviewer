package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	App               AppConfig
	AI                AIConfig
	Database          DatabaseConfig
	Log               LogConfig
	Dashboard         DashboardConfig
	ClientsConfigPath string
}

type AppConfig struct {
	Port string
	Env  string
}

type AIConfig struct {
	Provider              string
	BaseURL               string
	Token                 string
	Model                 string
	TimeoutSeconds        int
	PricePerMInputTokens  float64
	PricePerMOutputTokens float64
}

type DashboardConfig struct {
	User      string
	Password  string
	JWTSecret string
}

type DatabaseConfig struct {
	Path string
}

type LogConfig struct {
	Level string
}

func Load() (*Config, error) {
	aiTimeout, err := strconv.Atoi(getEnv("AI_TIMEOUT_SECONDS", "120"))
	if err != nil {
		return nil, fmt.Errorf("AI_TIMEOUT_SECONDS must be an integer: %w", err)
	}

	priceInput, err := strconv.ParseFloat(getEnv("AI_PRICE_PER_M_INPUT_TOKENS", "3.0"), 64)
	if err != nil {
		return nil, fmt.Errorf("AI_PRICE_PER_M_INPUT_TOKENS must be a number: %w", err)
	}

	priceOutput, err := strconv.ParseFloat(getEnv("AI_PRICE_PER_M_OUTPUT_TOKENS", "15.0"), 64)
	if err != nil {
		return nil, fmt.Errorf("AI_PRICE_PER_M_OUTPUT_TOKENS must be a number: %w", err)
	}

	aiProvider := strings.ToLower(getEnv("AI_PROVIDER", ""))
	if aiProvider == "" {
		if os.Getenv("GEMINI_API_KEY") != "" {
			aiProvider = "gemini"
		} else {
			aiProvider = "chat_completions"
		}
	}

	aiBaseURL, aiToken, aiModel, err := loadAIProviderConfig(aiProvider)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		App: AppConfig{
			Port: getEnv("APP_PORT", "8080"),
			Env:  getEnv("APP_ENV", "development"),
		},
		AI: AIConfig{
			Provider:              aiProvider,
			BaseURL:               aiBaseURL,
			Token:                 aiToken,
			Model:                 aiModel,
			TimeoutSeconds:        aiTimeout,
			PricePerMInputTokens:  priceInput,
			PricePerMOutputTokens: priceOutput,
		},
		Database: DatabaseConfig{
			Path: getEnv("DATABASE_PATH", "./data/reviewer.db"),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Dashboard: DashboardConfig{
			User:      getEnv("DASHBOARD_USER", "admin"),
			Password:  getEnv("DASHBOARD_PASSWORD", ""),
			JWTSecret: getEnv("DASHBOARD_JWT_SECRET", ""),
		},
		ClientsConfigPath: getEnv("CLIENTS_CONFIG_PATH", "./clients.json"),
	}

	return cfg, nil
}

func loadAIProviderConfig(provider string) (baseURL, token, model string, err error) {
	switch provider {
	case "gemini":
		return firstEnv("GEMINI_BASE_URL", "AI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta"),
			requireAnyEnv("GEMINI_API_KEY", "AI_TOKEN"),
			firstEnv("GEMINI_MODEL", "AI_MODEL", "gemini-flash-latest"),
			nil
	case "chat_completions", "chat-completions", "openai", "claude":
		return requireEnv("AI_BASE_URL"),
			requireEnv("AI_TOKEN"),
			getEnv("AI_MODEL", "claude-sonnet-4"),
			nil
	default:
		return "", "", "", fmt.Errorf("AI_PROVIDER must be gemini or chat_completions, got %q", provider)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstEnv(primary, fallbackKey, fallbackValue string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	return getEnv(fallbackKey, fallbackValue)
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func requireAnyEnv(keys ...string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	panic(fmt.Sprintf("required environment variable %q is not set", strings.Join(keys, " or ")))
}
