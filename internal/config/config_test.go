package config

import "testing"

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

	if cfg.AI.BaseURL != "https://generativelanguage.googleapis.com/v1beta" {
		t.Fatalf("BaseURL = %q, want Gemini default", cfg.AI.BaseURL)
	}
	if cfg.AI.Provider != "gemini" {
		t.Fatalf("Provider = %q, want gemini", cfg.AI.Provider)
	}
	if cfg.AI.Token != "gemini-key" {
		t.Fatalf("Token = %q, want GEMINI_API_KEY", cfg.AI.Token)
	}
	if cfg.AI.Model != "gemini-flash-latest" {
		t.Fatalf("Model = %q, want Gemini default", cfg.AI.Model)
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
	if cfg.AI.Token != "gemini-key" {
		t.Fatalf("Token = %q, want GEMINI_API_KEY", cfg.AI.Token)
	}
	if cfg.AI.Model != "gemini-custom" {
		t.Fatalf("Model = %q, want GEMINI_MODEL", cfg.AI.Model)
	}
}

func TestLoad_ChatCompletionsProvider(t *testing.T) {
	t.Setenv("AI_PROVIDER", "chat_completions")
	t.Setenv("AI_BASE_URL", "https://chat.example")
	t.Setenv("AI_TOKEN", "chat-token")
	t.Setenv("AI_MODEL", "claude-sonnet-4")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_BASE_URL", "")
	t.Setenv("GEMINI_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.AI.Provider != "chat_completions" {
		t.Fatalf("Provider = %q, want chat_completions", cfg.AI.Provider)
	}
	if cfg.AI.BaseURL != "https://chat.example" {
		t.Fatalf("BaseURL = %q, want AI_BASE_URL", cfg.AI.BaseURL)
	}
	if cfg.AI.Token != "chat-token" {
		t.Fatalf("Token = %q, want AI_TOKEN", cfg.AI.Token)
	}
	if cfg.AI.Model != "claude-sonnet-4" {
		t.Fatalf("Model = %q, want AI_MODEL", cfg.AI.Model)
	}
}
