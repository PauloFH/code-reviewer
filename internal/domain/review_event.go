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
	ID               int64        `json:"id"`
	ClientName       string       `json:"client_name"`
	ProjectRef       string       `json:"project_ref"`
	MRIID            int64        `json:"mr_iid"`
	MRID             int64        `json:"mr_id"`
	MRTitle          string       `json:"mr_title"`
	MRURL            string       `json:"mr_url"`
	AuthorUsername   string       `json:"author_username"`
	PromptTokens     int          `json:"prompt_tokens"`
	CompletionTokens int          `json:"completion_tokens"`
	TotalTokens      int          `json:"total_tokens"`
	CostUSD          float64      `json:"cost_usd"`
	DurationMS       int64        `json:"duration_ms"`
	Status           ReviewStatus `json:"status"`
	ErrorMessage     string       `json:"error_message,omitempty"`
	ReviewedAt       time.Time    `json:"reviewed_at"`
}

// ReviewEventFilter is used by the repository list method.
type ReviewEventFilter struct {
	ClientName string
	Search     string
	Since      time.Time
	Until      time.Time
	Limit      int // 0 → default 50
	Offset     int
}

// ReviewStats is a pre-aggregated summary for the dashboard.
type ReviewStats struct {
	TotalReviews   int64        `json:"total_reviews"`
	TotalTokens    int64        `json:"total_tokens"`
	TotalCostUSD   float64      `json:"total_cost_usd"`
	ReviewsLast30d int64        `json:"reviews_last_30d"`
	CostLast30d    float64      `json:"cost_last_30d"`
	ByClient       []ClientStat `json:"by_client"`
}

// ClientStat holds per-client aggregates inside ReviewStats.
type ClientStat struct {
	ClientName   string  `json:"client_name"`
	ReviewCount  int64   `json:"review_count"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
}

// DailyCostPoint is one data point in the 30-day cost chart.
type DailyCostPoint struct {
	Date    string  `json:"date"`
	CostUSD float64 `json:"cost_usd"`
	Reviews int64   `json:"reviews"`
}
