package usecase

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// EventPublisher feeds the dashboard live feed.
type EventPublisher interface {
	Publish(e domain.ReviewEvent)
}

// RepoFetcher checks out the MR working tree for agentic review.
type RepoFetcher interface {
	Checkout(src domain.RepoSource, sha string) (domain.CodeWorkspace, func(), error)
}

// ReviewDeps are the collaborators of the review pipeline; all are required.
type ReviewDeps struct {
	Registry domain.ClientRegistry
	Claims   domain.ReviewRepository
	Events   interface {
		SaveEvent(domain.ReviewEvent) error
	}
	Users interface {
		GetByUsername(string) (*domain.User, error)
	}
	Profiles interface {
		Resolve(clientName, projectRef string) (*domain.ReviewProfile, error)
	}
	Fetcher   RepoFetcher
	Reviewer  domain.AIReviewer
	Publisher EventPublisher
	Retry     RetryConfig
	Log       *slog.Logger
}

// ProcessMergeRequest reviews an MR once per watched user: it claims the MR,
// gathers the context the resolved profile asks for, has the AI review it,
// posts the result and records the outcome for the dashboard.
type ProcessMergeRequest struct {
	ReviewDeps
	related *RelatedCollector
}

func NewProcessMergeRequest(d ReviewDeps) *ProcessMergeRequest {
	return &ProcessMergeRequest{ReviewDeps: d, related: NewRelatedCollector(d.Log)}
}

// Execute runs the pipeline for one event. It returns nil when the event was
// reviewed or intentionally skipped.
func (uc *ProcessMergeRequest) Execute(event *domain.WebhookEvent) error {
	start := time.Now()

	log := uc.Log.With(
		slog.String("client", event.ClientName),
		slog.String("project", event.ProjectRef),
		slog.Int64("mr_iid", event.MRIID),
		slog.String("action", event.Action),
	)

	log.Info("webhook received")

	client, ok := uc.Registry.Get(event.ClientName)
	if !ok {
		return fmt.Errorf("client %q not found in registry", event.ClientName)
	}

	username := client.WatchedUsername()

	var mr *domain.MergeRequest
	err := uc.Retry.run(log, "fetching MR", func() error {
		var fetchErr error
		mr, fetchErr = client.GetMergeRequest(event.ProjectRef, event.MRIID)
		return fetchErr
	})
	if err != nil {
		log.Error("error fetching merge request from platform", slog.String("error", err.Error()))
		uc.saveEvent(event, mr, domain.ReviewStatusError, err.Error(), nil, start)
		// No failure note here: until the MR is read we don't know the watched
		// user is on it, and a comment could land on an unrelated MR.
		return fmt.Errorf("fetching MR: %w", err)
	}

	if !mr.IsUserAssigned(username) {
		log.Info("webhook ignored: watched user is not assigned", slog.String("username", username))
		uc.saveEvent(event, mr, domain.ReviewStatusSkipped, "", nil, start)
		return nil
	}

	log.Info("watched user is assigned", slog.String("username", username))

	// Only the MR author or a dashboard user may trigger a review from a comment.
	if event.IsReanalyze && event.CommentAuthor != "" && event.CommentAuthor != mr.AuthorUsername {
		_, err := uc.Users.GetByUsername(event.CommentAuthor)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			log.Error("checking /review author", slog.String("error", err.Error()))
		}
		if err != nil {
			log.Info("unauthorized /review command", slog.String("comment_author", event.CommentAuthor))
			uc.saveEvent(event, mr, domain.ReviewStatusSkipped, "unauthorized /review command", nil, start)
			return nil
		}
	}

	previousReview := ""
	previousHeadSHA := ""
	if event.IsReanalyze {
		var previous *domain.ReviewHistory
		if event.ReviewJustification != "" {
			previous, err = uc.Claims.GetReviewHistory(event.ClientName, event.ProjectRef, event.MRIID, username)
			if err != nil {
				log.Warn("failed to load previous review", slog.String("error", err.Error()))
			}
		}
		if err := uc.Claims.ClearReview(event.ClientName, event.ProjectRef, event.MRIID); err != nil {
			log.Error("failed to clear review record", slog.String("error", err.Error()))
		}
		if previous != nil {
			previousReview = previous.Body
			previousHeadSHA = previous.HeadSHA
		}
	}

	claimed, err := uc.Claims.ClaimReview(event.ClientName, event.ProjectRef, event.MRIID, mr.ID, username)
	if err != nil {
		log.Error("error claiming review", slog.String("error", err.Error()))
		return fmt.Errorf("claiming review: %w", err)
	}
	if !claimed {
		log.Info("MR already claimed, skipping duplicate", slog.String("username", username))
		uc.saveEvent(event, mr, domain.ReviewStatusSkipped, "already reviewed", nil, start)
		return nil
	}
	reviewPosted := false
	defer func() {
		if reviewPosted {
			return
		}
		if err := uc.Claims.ClearReview(event.ClientName, event.ProjectRef, event.MRIID); err != nil {
			log.Error("failed to release review claim", slog.String("error", err.Error()))
		}
	}()

	log.Info("fetching diff")

	var diff string
	err = uc.Retry.run(log, "fetching diff", func() error {
		var fetchErr error
		diff, fetchErr = client.GetMergeRequestDiff(event.ProjectRef, event.MRIID)
		return fetchErr
	})
	if err != nil {
		log.Error("error fetching diff from platform", slog.String("error", err.Error()))
		uc.saveEvent(event, mr, domain.ReviewStatusError, err.Error(), nil, start)
		uc.notifyFailure(log, client, event, "buscar o diff", err)
		return fmt.Errorf("fetching diff: %w", err)
	}
	// A justified re-review of new commits only needs what changed since the last review.
	if previousHeadSHA != "" && previousHeadSHA != mr.HeadSHA {
		if provider, ok := client.(domain.IncrementalDiffProvider); ok {
			if incremental, err := provider.GetMergeRequestDiffSince(event.ProjectRef, previousHeadSHA, mr.HeadSHA); err != nil {
				log.Warn("failed to fetch incremental diff", slog.String("error", err.Error()))
			} else {
				diff = incremental
			}
		}
	}

	profile := uc.resolveProfile(log, event)

	var related *domain.RelatedContext
	if profile.IncludeRelated {
		related = uc.related.Collect(client, mr)
		if related != nil {
			log.Info("related context collected",
				slog.Int("issues", len(related.Issues)),
				slog.Int("linked_mrs", len(related.LinkedMRs)),
				slog.Int("discussions", len(related.Discussions)),
			)
		}
	}

	var workspace domain.CodeWorkspace
	if profile.Agentic {
		if ws, cleanup, ok := uc.checkoutWorkspace(log, client, event, mr); ok {
			workspace = ws
			defer cleanup()
			log.Info("agentic workspace checked out", slog.String("root", ws.Root()))
		}
	}

	log.Info("sending diff to AI reviewer", slog.String("profile", profile.Name))

	var result *domain.ReviewResult
	err = uc.Retry.run(log, "generating review", func() error {
		var reviewErr error
		result, reviewErr = uc.Reviewer.ReviewDiff(domain.ReviewInput{
			MergeRequest:        mr,
			Diff:                diff,
			ClientName:          event.ClientName,
			ProjectRef:          event.ProjectRef,
			Profile:             &profile,
			Related:             related,
			Workspace:           workspace,
			PreviousReview:      previousReview,
			ReviewJustification: event.ReviewJustification,
		})
		return reviewErr
	})
	if err != nil {
		log.Error("AI reviewer returned error", slog.String("error", err.Error()))
		uc.saveEvent(event, mr, domain.ReviewStatusError, err.Error(), nil, start)
		uc.notifyFailure(log, client, event, "gerar a revisão com IA", err)
		return fmt.Errorf("generating review: %w", err)
	}

	if err := uc.Retry.run(log, "posting comment", func() error {
		return client.CreateMergeRequestNote(event.ProjectRef, event.MRIID, result.Body)
	}); err != nil {
		log.Error("error posting comment to platform", slog.String("error", err.Error()))
		uc.saveEvent(event, mr, domain.ReviewStatusError, err.Error(), result, start)
		uc.notifyFailure(log, client, event, "publicar o comentário de revisão", err)
		return fmt.Errorf("posting comment: %w", err)
	}
	reviewPosted = true
	if err := uc.Claims.CompleteReview(event.ClientName, event.ProjectRef, event.MRIID, username, mr.HeadSHA, result.Body); err != nil {
		log.Error("failed to save review history", slog.String("error", err.Error()))
	}

	log.Info("review comment posted",
		slog.String("mr_url", mr.WebURL),
		slog.Int("total_tokens", result.Usage.TotalTokens),
	)

	uc.postInlineComments(log, client, mr, result.InlineComments)

	uc.saveEvent(event, mr, domain.ReviewStatusSuccess, "", result, start)

	return nil
}

// notifyFailure posts a best-effort comment on the MR explaining that the
// automatic review could not be completed after the configured retries were
// exhausted at the given stage, and inviting a manual "/review" retry later.
// Failures posting the notice itself are only logged — the caller already has
// an error to return.
func (uc *ProcessMergeRequest) notifyFailure(log *slog.Logger, client domain.PlatformClient, event *domain.WebhookEvent, stage string, cause error) {
	body := fmt.Sprintf(
		"⚠️ Não foi possível concluir a revisão automática (%s) após %d tentativa(s): %s\n\nComente `/review` para tentar novamente mais tarde.",
		stage, uc.Retry.attempts(), cause.Error(),
	)
	if err := client.CreateMergeRequestNote(event.ProjectRef, event.MRIID, body); err != nil {
		log.Error("failed to post failure notice to MR", slog.String("error", err.Error()))
	}
}

func (uc *ProcessMergeRequest) postInlineComments(log *slog.Logger, client domain.PlatformClient, mr *domain.MergeRequest, comments []domain.InlineComment) {
	for _, comment := range comments {
		err := client.CreateMergeRequestInlineComment(mr, comment)
		if err == nil {
			log.Info("inline review comment posted",
				slog.String("path", comment.Path),
				slog.Int("line", comment.Line),
			)
			continue
		}
		if errors.Is(err, domain.ErrInlineCommentsUnsupported) {
			log.Warn("inline review comments are not supported by this platform",
				slog.String("platform", string(client.Platform())),
			)
			return
		}
		log.Error("error posting inline review comment",
			slog.String("path", comment.Path),
			slog.Int("line", comment.Line),
			slog.String("error", err.Error()),
		)
	}
}

// resolveProfile picks the review profile for this event: the one assigned to
// the most specific scope (project → client → global), or the diff-only default
// when nothing is assigned. A valid "/review <preset>" override replaces it.
func (uc *ProcessMergeRequest) resolveProfile(log *slog.Logger, event *domain.WebhookEvent) domain.ReviewProfile {
	profile := domain.DefaultReviewProfile()
	resolved, err := uc.Profiles.Resolve(event.ClientName, event.ProjectRef)
	if err != nil {
		log.Error("resolving review profile, using diff-only default", slog.String("error", err.Error()))
	} else if resolved != nil {
		profile = *resolved
	}
	if event.RequestedProfile != "" {
		if preset, ok := domain.ProfileFromPreset(event.RequestedProfile); ok {
			profile = preset
			log.Info("review profile overridden by /review command", slog.String("preset", event.RequestedProfile))
		} else {
			log.Warn("unknown /review preset, ignoring", slog.String("preset", event.RequestedProfile))
		}
	}
	return profile
}

// checkoutWorkspace clones the MR working tree for agentic review. Best-effort:
// returns ok=false (and the review proceeds without a workspace) on any problem.
func (uc *ProcessMergeRequest) checkoutWorkspace(log *slog.Logger, client domain.PlatformClient, event *domain.WebhookEvent, mr *domain.MergeRequest) (domain.CodeWorkspace, func(), bool) {
	locator, ok := client.(domain.RepoLocator)
	if !ok {
		log.Warn("agentic profile but platform has no RepoLocator; proceeding without workspace",
			slog.String("platform", string(client.Platform())))
		return nil, nil, false
	}
	src, err := locator.RepoSource(event.ProjectRef)
	if err != nil {
		log.Error("resolving repo source for agentic review", slog.String("error", err.Error()))
		return nil, nil, false
	}
	ws, cleanup, err := uc.Fetcher.Checkout(src, mr.HeadSHA)
	if err != nil {
		log.Error("checking out working tree, proceeding without workspace", slog.String("error", err.Error()))
		return nil, nil, false
	}
	return ws, cleanup, true
}

// saveEvent records the outcome for the dashboard; failures are only logged.
func (uc *ProcessMergeRequest) saveEvent(
	event *domain.WebhookEvent,
	mr *domain.MergeRequest,
	status domain.ReviewStatus,
	errMsg string,
	result *domain.ReviewResult,
	start time.Time,
) {
	e := domain.ReviewEvent{
		ClientName:   event.ClientName,
		ProjectRef:   event.ProjectRef,
		MRIID:        event.MRIID,
		Status:       status,
		ErrorMessage: errMsg,
		DurationMS:   time.Since(start).Milliseconds(),
		ReviewedAt:   time.Now().UTC(),
	}
	if result != nil {
		e.PromptTokens = result.Usage.PromptTokens
		e.CompletionTokens = result.Usage.CompletionTokens
		e.TotalTokens = result.Usage.TotalTokens
		e.CostUSD = result.CostUSD
	}
	if mr != nil {
		e.MRID = mr.ID
		e.MRTitle = mr.Title
		e.MRURL = mr.WebURL
		e.AuthorUsername = mr.AuthorUsername
	}
	if err := uc.Events.SaveEvent(e); err != nil {
		uc.Log.Error("failed to save review event", slog.String("error", err.Error()))
	}
	uc.Publisher.Publish(e)
}
