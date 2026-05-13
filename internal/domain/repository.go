package domain

type ReviewRepository interface {
	AlreadyReviewed(projectID, mrIID int64, username string) (bool, error)
	MarkAsReviewed(projectID, mrIID, mrID int64, username string) error
}

type GitLabClient interface {
	GetMergeRequest(projectID, mrIID int64) (*MergeRequest, error)
	GetMergeRequestDiff(projectID, mrIID int64) (string, error)
	CreateMergeRequestNote(projectID, mrIID int64, body string) error
}

type AIReviewer interface {
	ReviewDiff(input ReviewInput) (*ReviewResult, error)
}
