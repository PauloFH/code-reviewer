package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
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

// RetryConfig tunes how transient failures in external calls (platform API,
// AI reviewer) are retried before the review pipeline gives up on a step.
type RetryConfig struct {
	MaxAttempts      int
	BaseDelaySeconds int
}

type Config struct {
	App AppConfig
	// AI is the env-configured fallback used when no AIConfig is assigned in the DB.
	AI                domain.AIConfig
	Database          DatabaseConfig
	Log               LogConfig
	Dashboard         DashboardConfig
	Agentic           AgenticConfig
	Sandbox           SandboxConfig
	Retry             RetryConfig
	ClientsConfigPath string
	// AIConfigEncKey (base64, 32 bytes) encrypts the stored AI credentials and
	// client tokens/webhook secrets; required as soon as any is stored.
	AIConfigEncKey string
}

type AppConfig struct {
	Port string
	Env  string
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
	aiTimeout, err := strconv.Atoi(getEnv("AI_TIMEOUT_SECONDS", strconv.Itoa(domain.DefaultAITimeoutSeconds)))
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

	fallbackAI, err := loadAIProviderConfig(aiProvider)
	if err != nil {
		return nil, err
	}
	fallbackAI.TimeoutSeconds = aiTimeout
	fallbackAI.PricePerMInputTokens = priceInput
	fallbackAI.PricePerMOutputTokens = priceOutput

	cfg := &Config{
		App: AppConfig{
			Port: getEnv("APP_PORT", "8080"),
			Env:  getEnv("APP_ENV", "development"),
		},
		AI: fallbackAI,
		Database: DatabaseConfig{
			Path: getEnv("DATABASE_PATH", "./data/reviewer.db"),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Dashboard: DashboardConfig{
			User:      getEnv("DASHBOARD_USER", "admin"),
			Password:  getEnv("DASHBOARD_PASSWORD", "admin123"),
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
		Retry: RetryConfig{
			MaxAttempts:      getEnvInt("RETRY_MAX_ATTEMPTS", 3),
			BaseDelaySeconds: getEnvInt("RETRY_BASE_DELAY_SECONDS", 2),
		},
		ClientsConfigPath: getEnv("CLIENTS_CONFIG_PATH", "./clients.json"),
		AIConfigEncKey:    getEnv("AI_CONFIG_ENC_KEY", ""),
	}

	return cfg, nil
}

// loadAIProviderConfig maps AI_PROVIDER to an AIConfig. Every non-gemini alias
// speaks the OpenAI-compatible chat completions wire, so it becomes ProviderOpenAI
// (even "claude", which in env means a Claude model behind an OpenAI-compatible proxy).
// Empty BaseURL/Model fall back to the provider table defaults.
func loadAIProviderConfig(provider string) (domain.AIConfig, error) {
	cfg := domain.AIConfig{Name: "env", Mode: domain.AIModeAPI}
	var err error
	switch provider {
	case "gemini":
		cfg.Provider = domain.ProviderGemini
		cfg.BaseURL = firstEnv("GEMINI_BASE_URL", "AI_BASE_URL", "")
		cfg.Model = firstEnv("GEMINI_MODEL", "AI_MODEL", "")
		cfg.APIKey, err = requireEnv("GEMINI_API_KEY", "AI_TOKEN")
	case "chat_completions", "chat-completions", "openai", "claude":
		cfg.Provider = domain.ProviderOpenAI
		cfg.Model = getEnv("AI_MODEL", "claude-sonnet-4")
		if cfg.BaseURL, err = requireEnv("AI_BASE_URL"); err == nil {
			cfg.APIKey, err = requireEnv("AI_TOKEN")
		}
	default:
		return domain.AIConfig{}, fmt.Errorf("AI_PROVIDER must be gemini or chat_completions, got %q", provider)
	}
	return cfg, err
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

// requireEnv returns the first set variable among keys.
func requireEnv(keys ...string) (string, error) {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("required environment variable %s is not set", strings.Join(keys, " or "))
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
