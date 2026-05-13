package usecase

import (
	"math"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestCalculateCost_Zero(t *testing.T) {
	cost := CalculateCost(domain.TokenUsage{}, 3.0, 15.0)
	if cost != 0 {
		t.Errorf("expected 0, got %f", cost)
	}
}

func TestCalculateCost_InputOnly(t *testing.T) {
	usage := domain.TokenUsage{PromptTokens: 1_000_000}
	cost := CalculateCost(usage, 3.0, 15.0)
	if math.Abs(cost-3.0) > 1e-9 {
		t.Errorf("expected 3.0, got %f", cost)
	}
}

func TestCalculateCost_OutputOnly(t *testing.T) {
	usage := domain.TokenUsage{CompletionTokens: 1_000_000}
	cost := CalculateCost(usage, 3.0, 15.0)
	if math.Abs(cost-15.0) > 1e-9 {
		t.Errorf("expected 15.0, got %f", cost)
	}
}

func TestCalculateCost_Mixed(t *testing.T) {
	usage := domain.TokenUsage{PromptTokens: 500_000, CompletionTokens: 200_000}
	// input:  0.5M * 3.0  = 1.5
	// output: 0.2M * 15.0 = 3.0
	// total: 4.5
	cost := CalculateCost(usage, 3.0, 15.0)
	if math.Abs(cost-4.5) > 1e-9 {
		t.Errorf("expected 4.5, got %f", cost)
	}
}

func TestCalculateCost_CustomPrices(t *testing.T) {
	usage := domain.TokenUsage{PromptTokens: 1_000_000, CompletionTokens: 1_000_000}
	cost := CalculateCost(usage, 1.0, 2.0)
	if math.Abs(cost-3.0) > 1e-9 {
		t.Errorf("expected 3.0, got %f", cost)
	}
}
