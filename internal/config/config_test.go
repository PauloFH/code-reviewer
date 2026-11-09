package config

import (
	"strings"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestLoad_UsesGeminiDefaults(t *testing.T) {
	t.Setenv("AI_PROVIDER", "gemini")
	t.Setenv("GEMINI_API_KEY", "gemini-key")
	t.Setenv("AI_TOKEN", "")
	t.Setenv("GEMINI_BASE_URL", "")
	t.Setenv("AI_BASE_URL", "")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("AI_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got := cfg.AI.BaseURLOrDefault(); got != "https://generativelanguage.googleapis.com/v1beta" {
		t.Fatalf("BaseURL = %q, want Gemini default", got)
	}
	if cfg.AI.Provider != domain.ProviderGemini {
		t.Fatalf("Provider = %q, want gemini", cfg.AI.Provider)
	}
	if cfg.AI.APIKey != "gemini-key" {
		t.Fatalf("Token = %q, want GEMINI_API_KEY", cfg.AI.APIKey)
	}
	if got := cfg.AI.ModelOrDefault(); got != "gemini-flash-latest" {
		t.Fatalf("Model = %q, want Gemini default", got)
	}
	if cfg.AI.TimeoutSeconds != 120 || cfg.AI.PricePerMInputTokens != 3.0 || cfg.AI.PricePerMOutputTokens != 15.0 {
		t.Fatalf("env defaults not applied: %+v", cfg.AI)
	}
}

func TestLoad_GeminiEnvOverridesLegacyAIEnv(t *testing.T) {
	t.Setenv("AI_PROVIDER", "gemini")
	t.Setenv("GEMINI_API_KEY", "gemini-key")
	t.Setenv("AI_TOKEN", "ai-token")
	t.Setenv("GEMINI_BASE_URL", "https://gemini.example/v1beta")
	t.Setenv("AI_BASE_URL", "https://legacy.example")
	t.Setenv("GEMINI_MODEL", "gemini-custom")
	t.Setenv("AI_MODEL", "legacy-model")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.AI.BaseURL != "https://gemini.example/v1beta" {
		t.Fatalf("BaseURL = %q, want GEMINI_BASE_URL", cfg.AI.BaseURL)
	}
	if cfg.AI.APIKey != "gemini-key" {
		t.Fatalf("Token = %q, want GEMINI_API_KEY", cfg.AI.APIKey)
	}
	if cfg.AI.Model != "gemini-custom" {
		t.Fatalf("Model = %q, want GEMINI_MODEL", cfg.AI.Model)
	}
}

func TestLoad_ChatCompletionsProvider(t *testing.T) {
	t.Setenv("AI_PROVIDER", "chat_completions")
	t.Setenv("AI_BASE_URL", "https://chat.example")
	t.Setenv("AI_TOKEN", "chat-token")
	t.Setenv("AI_MODEL", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_BASE_URL", "")
	t.Setenv("GEMINI_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.AI.Provider != domain.ProviderOpenAI {
		t.Fatalf("Provider = %q, want openai (chat completions wire)", cfg.AI.Provider)
	}
	if cfg.AI.BaseURL != "https://chat.example" {
		t.Fatalf("BaseURL = %q, want AI_BASE_URL", cfg.AI.BaseURL)
	}
	if cfg.AI.APIKey != "chat-token" {
		t.Fatalf("Token = %q, want AI_TOKEN", cfg.AI.APIKey)
	}
	if cfg.AI.Model != "claude-sonnet-4" {
		t.Fatalf("Model = %q, want the chat completions env default", cfg.AI.Model)
	}
}

func TestLoad_ClaudeAliasUsesChatCompletionsWire(t *testing.T) {
	t.Setenv("AI_PROVIDER", "claude")
	t.Setenv("AI_BASE_URL", "https://proxy.example")
	t.Setenv("AI_TOKEN", "tok")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AI.Provider != domain.ProviderOpenAI || cfg.AI.BaseURL != "https://proxy.example" {
		t.Fatalf("claude alias must map to the openai-compatible wire, got %+v", cfg.AI)
	}
}

func TestLoad_RejectsUnknownProvider(t *testing.T) {
	t.Setenv("AI_PROVIDER", "mistral")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for unknown AI_PROVIDER")
	}
}

func TestLoadReportsMissingRequiredEnv(t *testing.T) {
	t.Setenv("AI_PROVIDER", "chat_completions")
	t.Setenv("AI_BASE_URL", "")
	t.Setenv("AI_TOKEN", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AI_BASE_URL") {
		t.Fatalf("Load() error = %v, want one naming AI_BASE_URL", err)
	}
}
