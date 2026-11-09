package domain

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

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// InlineComment is a review finding that can be posted on a changed line.
// Line refers to the new/right side line number in the MR/PR diff.
type InlineComment struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Body string `json:"body"`
}

// ReviewResult is a finished review. InlineComments are already validated
// against the diff the model saw, deduplicated and capped.
type ReviewResult struct {
	Body           string
	InlineComments []InlineComment
	Usage          TokenUsage
	CostUSD        float64
}

type ReviewHistory struct {
	HeadSHA string
	Body    string
}
