package domain

import "time"

type ReviewInput struct {
	MergeRequest *MergeRequest
	Diff         string
}

type ReviewResult struct {
	Body string
}

type ReviewedMergeRequest struct {
	ID           int64
	ProjectID    int64
	MRIID        int64
	MRID         int64
	ReviewedUser string
	ReviewedAt   time.Time
}
