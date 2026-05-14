package domain

import "time"

type ReviewInput struct {
	MergeRequest *MergeRequest
	Diff         string
	ClientName   string
	ProjectRef   string
}

// TokenUsage holds token consumption data returned by the AI API.
// Zero-value is safe — all mocks and tests that don't set it will produce $0 cost.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type ReviewResult struct {
	Body  string
	Usage TokenUsage
}

type ReviewedMergeRequest struct {
	ID           int64
	ProjectID    int64
	MRIID        int64
	MRID         int64
	ReviewedUser string
	ReviewedAt   time.Time
}
