package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	App      AppConfig
	GitLab   GitLabConfig
	AI       AIConfig
	Database DatabaseConfig
	Log      LogConfig
}

type AppConfig struct {
	Port string
	Env  string
}

type GitLabConfig struct {
	BaseURL       string
	Token         string
	WebhookSecret string
	Username      string
}

type AIConfig struct {
	BaseURL        string
	Token          string
	Model          string
	TimeoutSeconds int
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

	cfg := &Config{
		App: AppConfig{
			Port: getEnv("APP_PORT", "8080"),
			Env:  getEnv("APP_ENV", "development"),
		},
		GitLab: GitLabConfig{
			BaseURL:       requireEnv("GITLAB_BASE_URL"),
			Token:         requireEnv("GITLAB_TOKEN"),
			WebhookSecret: requireEnv("GITLAB_WEBHOOK_SECRET"),
			Username:      requireEnv("GITLAB_USERNAME"),
		},
		AI: AIConfig{
			BaseURL:        requireEnv("AI_BASE_URL"),
			Token:          requireEnv("AI_TOKEN"),
			Model:          getEnv("AI_MODEL", "claude-sonnet-4"),
			TimeoutSeconds: aiTimeout,
		},
		Database: DatabaseConfig{
			Path: getEnv("DATABASE_PATH", "./data/reviewer.db"),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}
