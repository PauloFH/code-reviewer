// Package github implements GitHub and the GitHub-compatible REST/webhook
// dialect that Gitea also speaks. projectRef is "owner/repo".
package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/forge"
)

type Client struct {
	forge.Identity
	api *forge.API
}

func New(c domain.Client, timeout time.Duration) *Client {
	return &Client{
		Identity: forge.Identity{Client: c},
		api: forge.NewAPI(c.BaseURL, timeout, func(req *http.Request) {
			req.Header.Set("Authorization", "Bearer "+c.Token)
			req.Header.Set("Accept", "application/vnd.github+json")
			req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		}),
	}
}

func (c *Client) ValidateWebhook(w domain.RawWebhook) bool {
	return forge.ValidHMAC(c.Client.WebhookSecret, w.Body, w.Header("X-Hub-Signature-256"), "sha256=")
}

func (c *Client) ParseWebhook(w domain.RawWebhook) (*domain.WebhookEvent, error) {
	return ParsePullRequestEvent(c.Client.Name, w.Header("X-GitHub-Event"), w.Body)
}

func (c *Client) GetMergeRequest(projectRef string, mrIID int64) (*domain.MergeRequest, error) {
	return GetPullRequest(c.api, projectRef, mrIID)
}

// GitHub serves a pull request's files in pages of at most 100 and stops
// listing after 3000 files.
const (
	filesPerPage = 100
	maxFilePages = 3000 / filesPerPage
)

func (c *Client) GetMergeRequestDiff(projectRef string, mrIID int64) (string, error) {
	var diff strings.Builder
	for page := 1; page <= maxFilePages; page++ {
		var files []struct {
			Filename string `json:"filename"`
			Patch    string `json:"patch"`
		}
		path := fmt.Sprintf("/repos/%s/pulls/%d/files?per_page=%d&page=%d", projectRef, mrIID, filesPerPage, page)
		if err := c.api.GetJSON(path, &files); err != nil {
			return "", fmt.Errorf("github GetMergeRequestDiff: %w", err)
		}
		for _, f := range files {
			if f.Patch != "" { // binary or too large for GitHub to inline
				diff.WriteString(forge.FileDiff(f.Filename, f.Filename, f.Patch))
			}
		}
		if len(files) < filesPerPage {
			break
		}
	}
	return diff.String(), nil
}

func (c *Client) CreateMergeRequestNote(projectRef string, mrIID int64, body string) error {
	return CreateIssueComment(c.api, projectRef, mrIID, body)
}

func (c *Client) CreateMergeRequestInlineComment(mr *domain.MergeRequest, comment domain.InlineComment) error {
	if mr.HeadSHA == "" {
		return fmt.Errorf("github CreateInlineComment: missing head SHA")
	}
	err := c.api.PostJSON(fmt.Sprintf("/repos/%s/pulls/%d/comments", mr.ProjectRef, mr.IID), map[string]any{
		"body":      comment.Body,
		"commit_id": mr.HeadSHA,
		"path":      comment.Path,
		"line":      comment.Line,
		"side":      "RIGHT",
	})
	if err != nil {
		return fmt.Errorf("github CreateInlineComment: %w", err)
	}
	return nil
}

// ParsePullRequestEvent normalises a pull_request webhook; eventType is the
// value of the platform's event header.
func ParsePullRequestEvent(clientName, eventType string, body []byte) (*domain.WebhookEvent, error) {
	if eventType != "pull_request" {
		return nil, fmt.Errorf("unsupported event %q", eventType)
	}
	var payload struct {
		Action     string `json:"action"`
		Number     int64  `json:"number"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parsing webhook body: %w", err)
	}
	return &domain.WebhookEvent{
		ClientName: clientName,
		ProjectRef: payload.Repository.FullName,
		MRIID:      payload.Number,
		Action:     payload.Action,
	}, nil
}

func GetPullRequest(api *forge.API, projectRef string, number int64) (*domain.MergeRequest, error) {
	if _, _, err := forge.SplitOwnerRepo(projectRef); err != nil {
		return nil, err
	}
	type login struct {
		Login string `json:"login"`
	}
	var raw struct {
		ID      int64  `json:"id"`
		Number  int64  `json:"number"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
		User    login  `json:"user"`
		Head    struct {
			SHA string `json:"sha"`
		} `json:"head"`
		RequestedReviewers []login `json:"requested_reviewers"`
		Assignees          []login `json:"assignees"`
	}
	if err := api.GetJSON(fmt.Sprintf("/repos/%s/pulls/%d", projectRef, number), &raw); err != nil {
		return nil, fmt.Errorf("GetMergeRequest: %w", err)
	}
	mr := &domain.MergeRequest{
		ID:             raw.ID,
		IID:            raw.Number,
		ProjectRef:     projectRef,
		Title:          raw.Title,
		Description:    raw.Body,
		AuthorUsername: raw.User.Login,
		State:          raw.State,
		WebURL:         raw.HTMLURL,
		HeadSHA:        raw.Head.SHA,
	}
	for _, r := range raw.RequestedReviewers {
		mr.Reviewers = append(mr.Reviewers, r.Login)
	}
	for _, a := range raw.Assignees {
		mr.Assignees = append(mr.Assignees, a.Login)
	}
	return mr, nil
}

// CreateIssueComment posts a conversation comment; the issues endpoint serves
// pull requests too.
func CreateIssueComment(api *forge.API, projectRef string, number int64, body string) error {
	if _, _, err := forge.SplitOwnerRepo(projectRef); err != nil {
		return err
	}
	if err := api.PostJSON(fmt.Sprintf("/repos/%s/issues/%d/comments", projectRef, number), map[string]string{"body": body}); err != nil {
		return fmt.Errorf("CreateNote: %w", err)
	}
	return nil
}
