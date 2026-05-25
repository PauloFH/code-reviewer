package domain

import "time"

type MergeRequest struct {
	ID             int64
	IID            int64
	ProjectRef     string // numeric string for GitLab, "owner/repo" for GitHub/Gitea
	Title          string
	Description    string
	AuthorUsername string
	Reviewers      []string
	Assignees      []string
	State          string
	WebURL         string
	HeadSHA        string
	DiffBaseSHA    string
	DiffStartSHA   string
	DiffHeadSHA    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (mr *MergeRequest) IsUserAssigned(username string) bool {
	for _, r := range mr.Reviewers {
		if r == username {
			return true
		}
	}
	for _, a := range mr.Assignees {
		if a == username {
			return true
		}
	}
	return false
}
