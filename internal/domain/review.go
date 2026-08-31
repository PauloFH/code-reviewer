package domain

import "time"

type ReviewInput struct {
	MergeRequest *MergeRequest
	Diff         string
	ClientName   string
	ProjectRef   string
	// Profile is the resolved review profile (layers). Nil means diff-only.
	Profile *ReviewProfile
	// Related is the gathered related context (issues, linked MRs, discussion).
	// Nil when the related layer is disabled or yielded nothing.
	Related *RelatedContext
	// Workspace is the checked-out working tree for agentic review.
	// Nil when the agentic layer is disabled.
	Workspace           CodeWorkspace
	PreviousReview      string
	ReviewJustification string
}

// TokenUsage holds token consumption data returned by the AI API.
// Zero-value is safe — all mocks and tests that don't set it will produce $0 cost.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// InlineComment is a review finding that can be posted on a changed line.
// Line refers to the new/right side line number in the MR/PR diff.
type InlineComment struct {
	Path string
	Line int
	Body string
}

type ReviewResult struct {
	Body           string
	InlineComments []InlineComment
	Usage          TokenUsage
	// Pricing of the AI config that produced this review, per million tokens.
	// Zero means "unknown" — the caller should fall back to its own defaults.
	PricePerMInputTokens  float64
	PricePerMOutputTokens float64
}

type ReviewHistory struct {
	HeadSHA string
	Body    string
}

type ReviewedMergeRequest struct {
	ID           int64
	ProjectID    int64
	MRIID        int64
	MRID         int64
	ReviewedUser string
	ReviewedAt   time.Time
}
