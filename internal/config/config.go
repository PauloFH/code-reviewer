package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AgenticConfig tunes the agentic review's repository checkout.
type AgenticConfig struct {
	WorkspaceDir    string // root for ephemeral checkouts
	CloneDepth      int    // shallow clone depth (0 = full clone)
	TimeoutSeconds  int    // total git operation timeout
	GitLabReposPath string // root of GitLab @hashed storage for same-machine local clones (empty = disabled)
}

// SandboxConfig tunes the execution sandbox. Image empty = sandbox disabled.
type SandboxConfig struct {
	Image          string
	CPUs           string
	Memory         string
	TimeoutSeconds int
}

type Config struct {
	App               AppConfig
	AI                AIConfig
	Database          DatabaseConfig
	Log               LogConfig
	Dashboard         DashboardConfig
	Agentic           AgenticConfig
	Sandbox           SandboxConfig
	ClientsConfigPath string
	// AIConfigEncKey is the base64-encoded 32-byte master key used to encrypt
	// frontend-managed AI credentials at rest. Required only when such configs
	// are persisted (env-based config does not use it).
	AIConfigEncKey string
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
		Agentic: AgenticConfig{
			WorkspaceDir:    getEnv("WORKSPACE_DIR", "./data/workspaces"),
			CloneDepth:      getEnvInt("AGENTIC_CLONE_DEPTH", 1),
			TimeoutSeconds:  getEnvInt("AGENTIC_TIMEOUT_SECONDS", 120),
			GitLabReposPath: getEnv("GITLAB_REPOS_PATH", ""),
		},
		Sandbox: SandboxConfig{
			Image:          getEnv("SANDBOX_IMAGE", ""),
			CPUs:           getEnv("SANDBOX_CPUS", "2"),
			Memory:         getEnv("SANDBOX_MEMORY", "2g"),
			TimeoutSeconds: getEnvInt("SANDBOX_TIMEOUT_SECONDS", 120),
		},
		ClientsConfigPath: getEnv("CLIENTS_CONFIG_PATH", "./clients.json"),
		AIConfigEncKey:    getEnv("AI_CONFIG_ENC_KEY", ""),
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

// getEnvInt reads an integer env var, returning fallback when unset or invalid.
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
