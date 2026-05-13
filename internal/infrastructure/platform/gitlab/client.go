package gitlab

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

// Client implements domain.PlatformClient for GitLab (cloud and self-hosted).
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

func (c *Client) Platform() domain.PlatformType { return domain.PlatformGitLab }
func (c *Client) ClientName() string             { return c.name }
func (c *Client) WatchedUsername() string        { return c.username }

// ---------- API methods ----------

func (c *Client) GetMergeRequest(projectRef string, mrIID int64) (*domain.MergeRequest, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d", c.baseURL, projectRef, mrIID)

	var raw struct {
		ID          int64  `json:"id"`
		IID         int64  `json:"iid"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       string `json:"state"`
		WebURL      string `json:"web_url"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
		Author      struct {
			Username string `json:"username"`
		} `json:"author"`
		Reviewers []struct {
			Username string `json:"username"`
		} `json:"reviewers"`
		Assignees []struct {
			Username string `json:"username"`
		} `json:"assignees"`
	}

	if err := c.get(url, &raw); err != nil {
		return nil, fmt.Errorf("gitlab GetMergeRequest: %w", err)
	}

	mr := &domain.MergeRequest{
		ID:             raw.ID,
		IID:            raw.IID,
		ProjectRef:     projectRef,
		Title:          raw.Title,
		Description:    raw.Description,
		AuthorUsername: raw.Author.Username,
		State:          raw.State,
		WebURL:         raw.WebURL,
	}
	for _, r := range raw.Reviewers {
		mr.Reviewers = append(mr.Reviewers, r.Username)
	}
	for _, a := range raw.Assignees {
		mr.Assignees = append(mr.Assignees, a.Username)
	}
	mr.CreatedAt, _ = time.Parse(time.RFC3339, raw.CreatedAt)
	mr.UpdatedAt, _ = time.Parse(time.RFC3339, raw.UpdatedAt)
	return mr, nil
}

func (c *Client) GetMergeRequestDiff(projectRef string, mrIID int64) (string, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/changes", c.baseURL, projectRef, mrIID)

	var raw struct {
		Changes []struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
			Diff    string `json:"diff"`
		} `json:"changes"`
	}

	if err := c.get(url, &raw); err != nil {
		return "", fmt.Errorf("gitlab GetMergeRequestDiff: %w", err)
	}

	var sb strings.Builder
	for _, ch := range raw.Changes {
		fmt.Fprintf(&sb, "--- a/%s\n+++ b/%s\n%s\n", ch.OldPath, ch.NewPath, ch.Diff)
	}
	return sb.String(), nil
}

func (c *Client) CreateMergeRequestNote(projectRef string, mrIID int64, body string) error {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/notes", c.baseURL, projectRef, mrIID)
	payload := map[string]string{"body": body}

	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab CreateNote: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitlab CreateNote: status %d: %s", resp.StatusCode, body)
	}
	return nil
}

// ---------- helpers ----------

func (c *Client) get(url string, out any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)

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
