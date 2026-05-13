package usecase

import "github.com/paulofh/code-reviewer/internal/domain"

// CalculateCost returns the USD cost for a given token usage based on per-million-token prices.
func CalculateCost(usage domain.TokenUsage, pricePerMInput, pricePerMOutput float64) float64 {
	input := float64(usage.PromptTokens) / 1_000_000 * pricePerMInput
	output := float64(usage.CompletionTokens) / 1_000_000 * pricePerMOutput
	return input + output
}
