package domain

// ReviewRepository tracks which MRs have already been reviewed per client,
// guaranteeing idempotency across service restarts.
type ReviewRepository interface {
	AlreadyReviewed(clientName, projectRef string, mrIID int64, username string) (bool, error)
	MarkAsReviewed(clientName, projectRef string, mrIID, mrID int64, username string) error
}

// AIReviewer sends a diff to an AI API and returns the review text.
type AIReviewer interface {
	ReviewDiff(input ReviewInput) (*ReviewResult, error)
}
