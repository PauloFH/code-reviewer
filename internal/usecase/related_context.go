package usecase

import (
	"log/slog"
	"regexp"
	"strconv"

	"github.com/paulofh/code-reviewer/internal/domain"
)

var (
	issueRefPattern = regexp.MustCompile(`(?:^|[^\w/])#(\d+)`)
	mrRefPattern    = regexp.MustCompile(`(?:^|[^\w/])!(\d+)`)
)

// detectIssueIIDs extracts GitLab-style issue references (#N) from text,
// returning unique IIDs in first-seen order.
func detectIssueIIDs(text string) []int64 {
	return uniqueIIDs(issueRefPattern, text)
}

// detectMRIIDs extracts GitLab-style merge request references (!N) from text.
func detectMRIIDs(text string) []int64 {
	return uniqueIIDs(mrRefPattern, text)
}

func uniqueIIDs(pattern *regexp.Regexp, text string) []int64 {
	var out []int64
	seen := map[int64]bool{}
	for _, m := range pattern.FindAllStringSubmatch(text, -1) {
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// RelatedCollector gathers issues, linked MRs and the discussion thread for an MR.
type RelatedCollector struct {
	log *slog.Logger
}

func NewRelatedCollector(log *slog.Logger) *RelatedCollector {
	return &RelatedCollector{log: log}
}

// Collect best-effort assembles the related context. Returns nil when the
// platform client does not support it or nothing was gathered.
func (rc *RelatedCollector) Collect(client domain.PlatformClient, mr *domain.MergeRequest) *domain.RelatedContext {
	provider, ok := client.(domain.RelatedContextProvider)
	if !ok {
		return nil
	}

	out := &domain.RelatedContext{}
	seenIssue := map[int64]bool{}

	// Issues the MR closes (most reliable source).
	if closes, err := provider.GetClosesIssues(mr.ProjectRef, mr.IID); err != nil {
		rc.log.Warn("fetching closes_issues", slog.String("error", err.Error()))
	} else {
		for _, is := range closes {
			if !seenIssue[is.IID] {
				seenIssue[is.IID] = true
				out.Issues = append(out.Issues, is)
			}
		}
	}

	// Issues referenced in the description (deduped against closes).
	for _, iid := range detectIssueIIDs(mr.Description) {
		if seenIssue[iid] {
			continue
		}
		is, err := provider.GetIssue(mr.ProjectRef, iid)
		if err != nil || is == nil {
			if err != nil {
				rc.log.Warn("fetching issue", slog.Int64("iid", iid), slog.String("error", err.Error()))
			}
			continue
		}
		seenIssue[iid] = true
		out.Issues = append(out.Issues, *is)
	}

	// Linked MRs referenced in the description (excluding the MR itself).
	for _, iid := range detectMRIIDs(mr.Description) {
		if iid == mr.IID {
			continue
		}
		linked, err := client.GetMergeRequest(mr.ProjectRef, iid)
		if err != nil || linked == nil {
			if err != nil {
				rc.log.Warn("fetching linked MR", slog.Int64("iid", iid), slog.String("error", err.Error()))
			}
			continue
		}
		out.LinkedMRs = append(out.LinkedMRs, domain.MergeRequestRef{
			IID: linked.IID, Title: linked.Title, State: linked.State, WebURL: linked.WebURL,
		})
	}

	// Discussion thread.
	if disc, err := provider.GetMergeRequestDiscussions(mr.ProjectRef, mr.IID); err != nil {
		rc.log.Warn("fetching discussions", slog.String("error", err.Error()))
	} else {
		out.Discussions = disc
	}

	if out.IsEmpty() {
		return nil
	}
	return out
}
