// Package gitlab implements GitLab (cloud and self-hosted). projectRef is the
// "group/repo" path or the numeric project id.
package gitlab

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/forge"
)

type Client struct {
	forge.Identity
	api       *forge.API
	reposPath string
}

// New builds a client; reposPath is the root of GitLab's @hashed repository
// storage when it lives on this machine (empty = clone over the network).
func New(c domain.Client, timeout time.Duration, reposPath string) *Client {
	return &Client{
		Identity: forge.Identity{Client: c},
		api: forge.NewAPI(c.BaseURL, timeout, func(req *http.Request) {
			req.Header.Set("PRIVATE-TOKEN", c.Token)
		}),
		reposPath: reposPath,
	}
}

func (c *Client) GetMergeRequest(projectRef string, mrIID int64) (*domain.MergeRequest, error) {
	var raw struct {
		ID          int64  `json:"id"`
		IID         int64  `json:"iid"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       string `json:"state"`
		WebURL      string `json:"web_url"`
		Author      user   `json:"author"`
		DiffRefs    struct {
			BaseSHA  string `json:"base_sha"`
			StartSHA string `json:"start_sha"`
			HeadSHA  string `json:"head_sha"`
		} `json:"diff_refs"`
		Reviewers []user `json:"reviewers"`
		Assignees []user `json:"assignees"`
	}
	if err := c.api.GetJSON(c.mrPath(projectRef, mrIID, ""), &raw); err != nil {
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
	}
	for _, r := range raw.Reviewers {
		mr.Reviewers = append(mr.Reviewers, r.Username)
	}
	for _, a := range raw.Assignees {
		mr.Assignees = append(mr.Assignees, a.Username)
	}
	return mr, nil
}

type user struct {
	Username string `json:"username"`
}

type fileChange struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
	Diff    string `json:"diff"`
}

func renderChanges(changes []fileChange) string {
	var sb strings.Builder
	for _, ch := range changes {
		sb.WriteString(forge.FileDiff(ch.OldPath, ch.NewPath, ch.Diff))
	}
	return sb.String()
}

func (c *Client) GetMergeRequestDiff(projectRef string, mrIID int64) (string, error) {
	var raw struct {
		Changes []fileChange `json:"changes"`
	}
	if err := c.api.GetJSON(c.mrPath(projectRef, mrIID, "/changes"), &raw); err != nil {
		return "", fmt.Errorf("gitlab GetMergeRequestDiff: %w", err)
	}
	return renderChanges(raw.Changes), nil
}

func (c *Client) GetMergeRequestDiffSince(projectRef, fromSHA, toSHA string) (string, error) {
	var raw struct {
		Diffs []fileChange `json:"diffs"`
	}
	path := fmt.Sprintf("%s/repository/compare?from=%s&to=%s", projectPath(projectRef), url.QueryEscape(fromSHA), url.QueryEscape(toSHA))
	if err := c.api.GetJSON(path, &raw); err != nil {
		return "", fmt.Errorf("gitlab GetMergeRequestDiffSince: %w", err)
	}
	return renderChanges(raw.Diffs), nil
}

func (c *Client) CreateMergeRequestNote(projectRef string, mrIID int64, body string) error {
	if err := c.api.PostJSON(c.mrPath(projectRef, mrIID, "/notes"), map[string]string{"body": body}); err != nil {
		return fmt.Errorf("gitlab CreateNote: %w", err)
	}
	return nil
}

func (c *Client) CreateMergeRequestInlineComment(mr *domain.MergeRequest, comment domain.InlineComment) error {
	if mr.DiffBaseSHA == "" || mr.DiffStartSHA == "" || mr.HeadSHA == "" {
		return fmt.Errorf("gitlab CreateInlineComment: missing diff refs")
	}
	form := url.Values{}
	form.Set("body", comment.Body)
	form.Set("position[position_type]", "text")
	form.Set("position[base_sha]", mr.DiffBaseSHA)
	form.Set("position[start_sha]", mr.DiffStartSHA)
	form.Set("position[head_sha]", mr.HeadSHA)
	form.Set("position[old_path]", comment.Path)
	form.Set("position[new_path]", comment.Path)
	form.Set("position[new_line]", strconv.Itoa(comment.Line))
	if err := c.api.PostForm(c.mrPath(mr.ProjectRef, mr.IID, "/discussions"), form); err != nil {
		return fmt.Errorf("gitlab CreateInlineComment: %w", err)
	}
	return nil
}

type issue struct {
	IID         int64  `json:"iid"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	WebURL      string `json:"web_url"`
}

func (i issue) toDomain() domain.Issue {
	return domain.Issue{IID: i.IID, Title: i.Title, Description: i.Description, State: i.State, WebURL: i.WebURL}
}

func (c *Client) GetIssue(projectRef string, issueIID int64) (*domain.Issue, error) {
	var raw issue
	if err := c.api.GetJSON(fmt.Sprintf("%s/issues/%d", projectPath(projectRef), issueIID), &raw); err != nil {
		return nil, fmt.Errorf("gitlab GetIssue: %w", err)
	}
	is := raw.toDomain()
	return &is, nil
}

func (c *Client) GetClosesIssues(projectRef string, mrIID int64) ([]domain.Issue, error) {
	var raw []issue
	if err := c.api.GetJSON(c.mrPath(projectRef, mrIID, "/closes_issues"), &raw); err != nil {
		return nil, fmt.Errorf("gitlab GetClosesIssues: %w", err)
	}
	out := make([]domain.Issue, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.toDomain())
	}
	return out, nil
}

func (c *Client) GetMergeRequestDiscussions(projectRef string, mrIID int64) ([]domain.Discussion, error) {
	var raw []struct {
		Notes []struct {
			Body   string `json:"body"`
			System bool   `json:"system"`
			Author user   `json:"author"`
		} `json:"notes"`
	}
	if err := c.api.GetJSON(c.mrPath(projectRef, mrIID, "/discussions"), &raw); err != nil {
		return nil, fmt.Errorf("gitlab GetMergeRequestDiscussions: %w", err)
	}
	var out []domain.Discussion
	for _, d := range raw {
		for _, n := range d.Notes {
			if n.System || n.Body == "" {
				continue // label changes and other system notes
			}
			out = append(out, domain.Discussion{Author: n.Author.Username, Body: n.Body})
		}
	}
	return out, nil
}

var (
	_ domain.RelatedContextProvider  = (*Client)(nil)
	_ domain.IncrementalDiffProvider = (*Client)(nil)
	_ domain.RepoLocator             = (*Client)(nil)
)

// hashedRepoPath computes GitLab's @hashed storage path for a project id:
// <root>/@hashed/<h[0:2]>/<h[2:4]>/<h>.git where h = hex(sha256(decimal id)).
func hashedRepoPath(root string, projectID int64) string {
	sum := sha256.Sum256([]byte(strconv.FormatInt(projectID, 10)))
	h := hex.EncodeToString(sum[:])
	return filepath.Join(root, "@hashed", h[0:2], h[2:4], h+".git")
}

func (c *Client) RepoSource(projectRef string) (domain.RepoSource, error) {
	u, err := url.Parse(c.api.BaseURL)
	if err != nil {
		return domain.RepoSource{}, fmt.Errorf("gitlab RepoSource: parsing base URL: %w", err)
	}

	path := projectRef
	var projectID int64
	// The clone URL needs the path (a numeric ref has none) and @hashed needs the id.
	if !strings.Contains(projectRef, "/") || c.reposPath != "" {
		var raw struct {
			ID                int64  `json:"id"`
			PathWithNamespace string `json:"path_with_namespace"`
		}
		if err := c.api.GetJSON(projectPath(projectRef), &raw); err != nil {
			return domain.RepoSource{}, fmt.Errorf("gitlab RepoSource: resolving project: %w", err)
		}
		if raw.PathWithNamespace != "" {
			path = raw.PathWithNamespace
		}
		projectID = raw.ID
	}

	u.Path = "/" + strings.TrimPrefix(path, "/") + ".git"
	src := domain.RepoSource{
		RemoteURL:  u.String(),
		AuthHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("oauth2:"+c.Client.Token)),
	}
	if c.reposPath != "" && projectID != 0 {
		src.LocalPath = hashedRepoPath(c.reposPath, projectID)
	}
	return src, nil
}

func (c *Client) mrPath(projectRef string, mrIID int64, suffix string) string {
	return fmt.Sprintf("%s/merge_requests/%d%s", projectPath(projectRef), mrIID, suffix)
}

// projectPath is the API path of a project; GitLab accepts a numeric id or a
// URL-encoded "group/repo".
func projectPath(ref string) string {
	return "/api/v4/projects/" + url.PathEscape(ref)
}
