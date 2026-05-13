package domain

import "time"

// ReviewStatus represents the outcome of processing a webhook event.
type ReviewStatus string

const (
	ReviewStatusSuccess ReviewStatus = "success"
	ReviewStatusSkipped ReviewStatus = "skipped"
	ReviewStatusError   ReviewStatus = "error"
)

// ReviewEvent is a full audit record of a single webhook processing pipeline.
type ReviewEvent struct {
	ID               int64
	ClientName       string
	ProjectRef       string
	MRIID            int64
	MRID             int64
	MRTitle          string
	MRURL            string
	AuthorUsername   string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
	DurationMS       int64
	Status           ReviewStatus
	ErrorMessage     string
	ReviewedAt       time.Time
}

// ReviewEventFilter is used by the repository list method.
type ReviewEventFilter struct {
	ClientName string
	Since      time.Time
	Until      time.Time
	Limit      int // 0 → default 50
	Offset     int
}

// ReviewStats is a pre-aggregated summary for the dashboard.
type ReviewStats struct {
	TotalReviews   int64
	TotalTokens    int64
	TotalCostUSD   float64
	ReviewsLast30d int64
	CostLast30d    float64
	ByClient       []ClientStat
}

// ClientStat holds per-client aggregates inside ReviewStats.
type ClientStat struct {
	ClientName   string
	ReviewCount  int64
	TotalTokens  int64
	TotalCostUSD float64
}
