package domain

import (
	"errors"
	"testing"
)

func TestAIConfigValidate(t *testing.T) {
	ok := AIConfig{Name: "c", Provider: ProviderClaude, Mode: AIModeAccount}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	cases := map[string]AIConfig{
		"no name":           {Provider: ProviderOpenAI, Mode: AIModeAPI},
		"unknown provider":  {Name: "c", Provider: "mistral", Mode: AIModeAPI},
		"unknown mode":      {Name: "c", Provider: ProviderOpenAI, Mode: "oauth"},
		"account on gemini": {Name: "c", Provider: ProviderGemini, Mode: AIModeAccount},
	}
	for name, c := range cases {
		var invalid *InvalidError
		if err := c.Validate(); !errors.As(err, &invalid) {
			t.Errorf("%s: err = %v, want *InvalidError", name, err)
		}
	}
}

func TestAIConfigDefaultsComeFromProviderTable(t *testing.T) {
	c := AIConfig{Provider: ProviderDeepSeek}
	if c.BaseURLOrDefault() != "https://api.deepseek.com/v1" || c.ModelOrDefault() != "deepseek-chat" {
		t.Fatalf("defaults = (%q, %q)", c.BaseURLOrDefault(), c.ModelOrDefault())
	}
	c.BaseURL, c.Model = "https://x", "m"
	if c.BaseURLOrDefault() != "https://x" || c.ModelOrDefault() != "m" {
		t.Fatal("explicit values must win over defaults")
	}
}
