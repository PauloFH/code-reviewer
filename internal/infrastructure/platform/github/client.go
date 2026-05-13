package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// Client implements domain.PlatformClient for GitHub (cloud and GitHub Enterprise).
// projectRef must be "owner/repo".
type Client struct {
	name     string
	baseURL  string
	token    string
	secret   string
	username string
	http     *http.Client
}

func New(name, baseURL, token, secret, username string, timeout time.Duration) *Client {
	return &Client{
		name:     name,
		baseURL:  strings.TrimRight(baseURL, "/"),
		token:    token,
		secret:   secret,
		username: username,
		http:     &http.Client{Timeout: timeout},
	}
}

func (c *Client) Platform() domain.PlatformType { return domain.PlatformGitHub }
func (c *Client) ClientName() string             { return c.name }
func (c *Client) WatchedUsername() string        { return c.username }

// ---------- API methods ----------

func (c *Client) GetMergeRequest(projectRef string, mrIID int64) (*domain.MergeRequest, error) {
	owner, repo, err := splitRef(projectRef)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.baseURL, owner, repo, mrIID)

	var raw struct {
		ID     int64  `json:"id"`
		Number int64  `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		State  string `json:"state"`
		HTMLURL string `json:"html_url"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		RequestedReviewers []struct {
			Login string `json:"login"`
		} `json:"requested_reviewers"`
		Assignees []struct {
			Login string `json:"login"`
		} `json:"assignees"`
	}

	if err := c.get(url, &raw); err != nil {
		return nil, fmt.Errorf("github GetMergeRequest: %w", err)
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
	}
	for _, r := range raw.RequestedReviewers {
		mr.Reviewers = append(mr.Reviewers, r.Login)
	}
	for _, a := range raw.Assignees {
		mr.Assignees = append(mr.Assignees, a.Login)
	}
	mr.CreatedAt, _ = time.Parse(time.RFC3339, raw.CreatedAt)
	mr.UpdatedAt, _ = time.Parse(time.RFC3339, raw.UpdatedAt)
	return mr, nil
}

func (c *Client) GetMergeRequestDiff(projectRef string, mrIID int64) (string, error) {
	owner, repo, err := splitRef(projectRef)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files", c.baseURL, owner, repo, mrIID)

	var files []struct {
		Filename string `json:"filename"`
		Patch    string `json:"patch"`
	}

	if err := c.get(url, &files); err != nil {
		return "", fmt.Errorf("github GetMergeRequestDiff: %w", err)
	}

	var sb strings.Builder
	for _, f := range files {
		if f.Patch == "" {
			continue
		}
		fmt.Fprintf(&sb, "--- a/%s\n+++ b/%s\n%s\n", f.Filename, f.Filename, f.Patch)
	}
	return sb.String(), nil
}

func (c *Client) CreateMergeRequestNote(projectRef string, mrIID int64, body string) error {
	owner, repo, err := splitRef(projectRef)
	if err != nil {
		return err
	}
	// GitHub: issue comments endpoint works for both issues and PRs
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, mrIID)
	payload := map[string]string{"body": body}

	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("github CreateNote: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github CreateNote: status %d: %s", resp.StatusCode, body)
	}
	return nil
}

// ---------- helpers ----------

func splitRef(ref string) (owner, repo string, err error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("github: invalid project ref %q (expected owner/repo)", ref)
	}
	return parts[0], parts[1], nil
}

func (c *Client) get(url string, out any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}
