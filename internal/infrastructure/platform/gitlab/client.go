package gitlab

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// Client implements domain.PlatformClient for GitLab (cloud and self-hosted).
type Client struct {
	platformType domain.PlatformType
	name         string
	baseURL      string
	token        string
	secret       string
	username     string
	reposPath    string
	http         *http.Client
}

func New(name, baseURL, token, secret, username string, timeout time.Duration, reposPath string) *Client {
	return newClient(domain.PlatformGitLab, name, baseURL, token, secret, username, timeout, reposPath)
}

func NewSelfHosted(name, baseURL, token, secret, username string, timeout time.Duration, reposPath string) *Client {
	return newClient(domain.PlatformGitLabSelfHosted, name, baseURL, token, secret, username, timeout, reposPath)
}

func newClient(pt domain.PlatformType, name, baseURL, token, secret, username string, timeout time.Duration, reposPath string) *Client {
	return &Client{
		platformType: pt,
		name:         name,
		baseURL:      strings.TrimRight(baseURL, "/"),
		token:        token,
		secret:       secret,
		username:     username,
		reposPath:    reposPath,
		http:         &http.Client{Timeout: timeout},
	}
}

func (c *Client) Platform() domain.PlatformType { return c.platformType }
func (c *Client) ClientName() string            { return c.name }
func (c *Client) WatchedUsername() string       { return c.username }

// ---------- API methods ----------

func (c *Client) GetMergeRequest(projectRef string, mrIID int64) (*domain.MergeRequest, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d", c.baseURL, encodeProjectRef(projectRef), mrIID)

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
		DiffRefs struct {
			BaseSHA  string `json:"base_sha"`
			StartSHA string `json:"start_sha"`
			HeadSHA  string `json:"head_sha"`
		} `json:"diff_refs"`
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
		HeadSHA:        raw.DiffRefs.HeadSHA,
		DiffBaseSHA:    raw.DiffRefs.BaseSHA,
		DiffStartSHA:   raw.DiffRefs.StartSHA,
		DiffHeadSHA:    raw.DiffRefs.HeadSHA,
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
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/changes", c.baseURL, encodeProjectRef(projectRef), mrIID)

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

func (c *Client) GetMergeRequestDiffSince(projectRef, fromSHA, toSHA string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v4/projects/%s/repository/compare?from=%s&to=%s", c.baseURL, encodeProjectRef(projectRef), url.QueryEscape(fromSHA), url.QueryEscape(toSHA))
	var raw struct {
		Diffs []struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
			Diff    string `json:"diff"`
		} `json:"diffs"`
	}
	if err := c.get(endpoint, &raw); err != nil {
		return "", fmt.Errorf("gitlab GetMergeRequestDiffSince: %w", err)
	}
	var out strings.Builder
	for _, diff := range raw.Diffs {
		fmt.Fprintf(&out, "--- a/%s\n+++ b/%s\n%s\n", diff.OldPath, diff.NewPath, diff.Diff)
	}
	return out.String(), nil
}

func (c *Client) CreateMergeRequestNote(projectRef string, mrIID int64, body string) error {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/notes", c.baseURL, encodeProjectRef(projectRef), mrIID)
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

func (c *Client) CreateMergeRequestInlineComment(projectRef string, mr *domain.MergeRequest, comment domain.InlineComment) error {
	if mr.DiffBaseSHA == "" || mr.DiffStartSHA == "" || mr.DiffHeadSHA == "" {
		return fmt.Errorf("gitlab CreateInlineComment: missing diff refs")
	}

	endpoint := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/discussions", c.baseURL, encodeProjectRef(projectRef), mr.IID)

	form := url.Values{}
	form.Set("body", comment.Body)
	form.Set("position[position_type]", "text")
	form.Set("position[base_sha]", mr.DiffBaseSHA)
	form.Set("position[start_sha]", mr.DiffStartSHA)
	form.Set("position[head_sha]", mr.DiffHeadSHA)
	form.Set("position[old_path]", comment.Path)
	form.Set("position[new_path]", comment.Path)
	form.Set("position[new_line]", fmt.Sprintf("%d", comment.Line))

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab CreateInlineComment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitlab CreateInlineComment: status %d: %s", resp.StatusCode, body)
	}

	return nil
}

// ---------- RelatedContextProvider ----------

func (c *Client) GetIssue(projectRef string, issueIID int64) (*domain.Issue, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/issues/%d", c.baseURL, encodeProjectRef(projectRef), issueIID)

	var raw struct {
		IID         int64  `json:"iid"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       string `json:"state"`
		WebURL      string `json:"web_url"`
	}
	if err := c.get(url, &raw); err != nil {
		return nil, fmt.Errorf("gitlab GetIssue: %w", err)
	}
	return &domain.Issue{
		IID: raw.IID, Title: raw.Title, Description: raw.Description,
		State: raw.State, WebURL: raw.WebURL,
	}, nil
}

func (c *Client) GetClosesIssues(projectRef string, mrIID int64) ([]domain.Issue, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/closes_issues", c.baseURL, encodeProjectRef(projectRef), mrIID)

	var raw []struct {
		IID         int64  `json:"iid"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       string `json:"state"`
		WebURL      string `json:"web_url"`
	}
	if err := c.get(url, &raw); err != nil {
		return nil, fmt.Errorf("gitlab GetClosesIssues: %w", err)
	}
	out := make([]domain.Issue, 0, len(raw))
	for _, r := range raw {
		out = append(out, domain.Issue{
			IID: r.IID, Title: r.Title, Description: r.Description, State: r.State, WebURL: r.WebURL,
		})
	}
	return out, nil
}

func (c *Client) GetMergeRequestDiscussions(projectRef string, mrIID int64) ([]domain.Discussion, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/discussions", c.baseURL, encodeProjectRef(projectRef), mrIID)

	var raw []struct {
		Notes []struct {
			Body   string `json:"body"`
			System bool   `json:"system"`
			Author struct {
				Username string `json:"username"`
			} `json:"author"`
		} `json:"notes"`
	}
	if err := c.get(url, &raw); err != nil {
		return nil, fmt.Errorf("gitlab GetMergeRequestDiscussions: %w", err)
	}
	var out []domain.Discussion
	for _, d := range raw {
		for _, n := range d.Notes {
			if n.System || n.Body == "" {
				continue // skip system notes (label changes, etc.)
			}
			out = append(out, domain.Discussion{Author: n.Author.Username, Body: n.Body})
		}
	}
	return out, nil
}

// Compile-time check: GitLab supports the optional related-context capability.
var _ domain.RelatedContextProvider = (*Client)(nil)
var _ domain.IncrementalDiffProvider = (*Client)(nil)

// ---------- RepoLocator ----------

// hashedRepoPath computes the GitLab @hashed storage path for a project id:
// <root>/@hashed/<h[0:2]>/<h[2:4]>/<h>.git where h = hex(sha256(decimal id)).
func hashedRepoPath(root string, projectID int64) string {
	sum := sha256.Sum256([]byte(strconv.FormatInt(projectID, 10)))
	h := hex.EncodeToString(sum[:])
	return filepath.Join(root, "@hashed", h[0:2], h[2:4], h+".git")
}

// RepoSource builds an authenticated HTTPS clone URL for the project and, when
// GITLAB_REPOS_PATH is configured, the local @hashed bare-repo path for fast
// same-machine clones. Resolves the project id/path via the API as needed.
func (c *Client) RepoSource(projectRef string) (domain.RepoSource, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return domain.RepoSource{}, fmt.Errorf("gitlab RepoSource: parsing base URL: %w", err)
	}

	path := projectRef
	var projectID int64

	// We need the API when the ref is numeric (to get the path) or when a local
	// repos path is configured (to get the id for @hashed).
	if !strings.Contains(projectRef, "/") || c.reposPath != "" {
		var raw struct {
			ID                int64  `json:"id"`
			PathWithNamespace string `json:"path_with_namespace"`
		}
		apiURL := fmt.Sprintf("%s/api/v4/projects/%s", c.baseURL, encodeProjectRef(projectRef))
		if err := c.get(apiURL, &raw); err != nil {
			return domain.RepoSource{}, fmt.Errorf("gitlab RepoSource: resolving project: %w", err)
		}
		if raw.PathWithNamespace != "" {
			path = raw.PathWithNamespace
		}
		projectID = raw.ID
	}

	u.User = url.UserPassword("oauth2", c.token)
	u.Path = "/" + strings.TrimPrefix(path, "/") + ".git"

	src := domain.RepoSource{RemoteURL: u.String()}
	if c.reposPath != "" && projectID != 0 {
		src.LocalPath = hashedRepoPath(c.reposPath, projectID)
	}
	return src, nil
}

// Compile-time check: GitLab can locate a repository for agentic checkout.
var _ domain.RepoLocator = (*Client)(nil)

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

// encodeProjectRef URL-encodes a project path (e.g. "group/subgroup/repo" → "group%2Fsubgroup%2Frepo").
// Numeric IDs pass through unchanged.
func encodeProjectRef(ref string) string {
	if strings.Contains(ref, "/") {
		return url.PathEscape(ref)
	}
	return ref
}
