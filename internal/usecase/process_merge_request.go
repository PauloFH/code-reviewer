package usecase

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// EventPublisher is a subset of the SSE broadcaster interface.
// Defined here to keep the usecase free of transport-layer imports.
type EventPublisher interface {
	Publish(e domain.ReviewEvent)
}

// ProcessMergeRequest is the central usecase. It receives a normalised
// WebhookEvent from the transport layer and orchestrates:
//
//  1. Resolve the platform client from the registry
//  2. Fetch full MR details
//  3. Verify the watched user was assigned/requested as reviewer
//  4. Check idempotency — skip if already reviewed
//  5. Fetch the MR diff
//  6. Send diff to the AI reviewer
//  7. Post the review as a comment on the MR
//  8. Persist the record to prevent future duplicates
//  9. Save a ReviewEvent for analytics (best-effort)
type ProcessMergeRequest struct {
	registry       domain.ClientRegistry
	repository     domain.ReviewRepository
	eventRepo      domain.ReviewEventRepository
	userRepo       domain.UserRepository
	reviewer       domain.AIReviewer
	publisher      EventPublisher
	pricePerMInput float64
	pricePerMOut   float64
	log            *slog.Logger
}

func NewProcessMergeRequest(
	registry domain.ClientRegistry,
	repository domain.ReviewRepository,
	eventRepo domain.ReviewEventRepository,
	userRepo domain.UserRepository,
	reviewer domain.AIReviewer,
	publisher EventPublisher,
	pricePerMInput, pricePerMOut float64,
	log *slog.Logger,
) *ProcessMergeRequest {
	return &ProcessMergeRequest{
		registry:       registry,
		repository:     repository,
		eventRepo:      eventRepo,
		userRepo:       userRepo,
		reviewer:       reviewer,
		publisher:      publisher,
		pricePerMInput: pricePerMInput,
		pricePerMOut:   pricePerMOut,
		log:            log,
	}
}

// Execute runs the full review pipeline for a single webhook event.
// Returns nil when the event was processed (or intentionally skipped).
// Returns a non-nil error only when a recoverable step failed and the
// caller should signal an error back to the platform.
func (uc *ProcessMergeRequest) Execute(event *domain.WebhookEvent) error {
	start := time.Now()

	log := uc.log.With(
		slog.String("client", event.ClientName),
		slog.String("project", event.ProjectRef),
		slog.Int64("mr_iid", event.MRIID),
		slog.String("action", event.Action),
	)

	log.Info("webhook received")

	client, ok := uc.registry.Get(event.ClientName)
	if !ok {
		return fmt.Errorf("client %q not found in registry", event.ClientName)
	}

	username := client.WatchedUsername()

	mr, err := client.GetMergeRequest(event.ProjectRef, event.MRIID)
	if err != nil {
		log.Error("error fetching merge request from platform", slog.String("error", err.Error()))
		uc.saveEvent(event, mr, domain.ReviewStatusError, err.Error(), nil, start)
		return fmt.Errorf("fetching MR: %w", err)
	}

	if !mr.IsUserAssigned(username) {
		log.Info("webhook ignored: watched user is not assigned", slog.String("username", username))
		uc.saveEvent(event, mr, domain.ReviewStatusSkipped, "", nil, start)
		return nil
	}

	log.Info("watched user is assigned", slog.String("username", username))

	// Validate re-analyze permission from MR comment
	if event.IsReanalyze && event.CommentAuthor != "" {
		isAuthor := event.CommentAuthor == mr.AuthorUsername
		isRegistered, _ := uc.userRepo.UserExists(event.CommentAuthor)
		if !isAuthor && !isRegistered {
			log.Info("unauthorized /review command", slog.String("comment_author", event.CommentAuthor))
			uc.saveEvent(event, mr, domain.ReviewStatusSkipped, "unauthorized /review command", nil, start)
			return nil
		}
	}

	// Re-analyze: clear idempotency record
	if event.IsReanalyze {
		if err := uc.repository.ClearReview(event.ClientName, event.ProjectRef, event.MRIID); err != nil {
			log.Error("failed to clear review record", slog.String("error", err.Error()))
		}
	}

	already, err := uc.repository.AlreadyReviewed(event.ClientName, event.ProjectRef, event.MRIID, username)
	if err != nil {
		log.Error("error checking idempotency", slog.String("error", err.Error()))
		return fmt.Errorf("checking idempotency: %w", err)
	}
	if already {
		log.Info("MR already reviewed, skipping duplicate", slog.String("username", username))
		uc.saveEvent(event, mr, domain.ReviewStatusSkipped, "already reviewed", nil, start)
		return nil
	}

	log.Info("fetching diff")

	diff, err := client.GetMergeRequestDiff(event.ProjectRef, event.MRIID)
	if err != nil {
		log.Error("error fetching diff from platform", slog.String("error", err.Error()))
		uc.saveEvent(event, mr, domain.ReviewStatusError, err.Error(), nil, start)
		return fmt.Errorf("fetching diff: %w", err)
	}

	log.Info("sending diff to AI reviewer")

	result, err := uc.reviewer.ReviewDiff(domain.ReviewInput{
		MergeRequest: mr,
		Diff:         diff,
		ClientName:   event.ClientName,
		ProjectRef:   event.ProjectRef,
	})
	if err != nil {
		log.Error("AI reviewer returned error", slog.String("error", err.Error()))
		uc.saveEvent(event, mr, domain.ReviewStatusError, err.Error(), nil, start)
		return fmt.Errorf("generating review: %w", err)
	}

	if err := client.CreateMergeRequestNote(event.ProjectRef, event.MRIID, result.Body); err != nil {
		log.Error("error posting comment to platform", slog.String("error", err.Error()))
		uc.saveEvent(event, mr, domain.ReviewStatusError, err.Error(), result, start)
		return fmt.Errorf("posting comment: %w", err)
	}

	log.Info("review comment posted",
		slog.String("mr_url", mr.WebURL),
		slog.Int("total_tokens", result.Usage.TotalTokens),
	)

	uc.postInlineComments(log, client, event.ProjectRef, mr, diff, result.InlineComments)

	// Best-effort idempotency record.
	if err := uc.repository.MarkAsReviewed(event.ClientName, event.ProjectRef, event.MRIID, mr.ID, username); err != nil {
		log.Error("error persisting review record — duplicate comments possible on retry",
			slog.String("error", err.Error()))
	}

	uc.saveEvent(event, mr, domain.ReviewStatusSuccess, "", result, start)

	return nil
}

func (uc *ProcessMergeRequest) postInlineComments(
	log *slog.Logger,
	client domain.PlatformClient,
	projectRef string,
	mr *domain.MergeRequest,
	diff string,
	comments []domain.InlineComment,
) {
	comments = filterInlineComments(comments, diff)
	if len(comments) == 0 {
		return
	}

	for _, comment := range comments {
		err := client.CreateMergeRequestInlineComment(projectRef, mr, comment)
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

// saveEvent persists a ReviewEvent and publishes it to the SSE broadcaster.
// Both operations are best-effort — failures are logged but never propagate.
func (uc *ProcessMergeRequest) saveEvent(
	event *domain.WebhookEvent,
	mr *domain.MergeRequest,
	status domain.ReviewStatus,
	errMsg string,
	result *domain.ReviewResult,
	start time.Time,
) {
	var usage domain.TokenUsage
	priceIn, priceOut := uc.pricePerMInput, uc.pricePerMOut
	if result != nil {
		usage = result.Usage
		if result.PricePerMInputTokens > 0 {
			priceIn = result.PricePerMInputTokens
		}
		if result.PricePerMOutputTokens > 0 {
			priceOut = result.PricePerMOutputTokens
		}
	}

	e := domain.ReviewEvent{
		ClientName:       event.ClientName,
		ProjectRef:       event.ProjectRef,
		MRIID:            event.MRIID,
		Status:           status,
		ErrorMessage:     errMsg,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CostUSD:          CalculateCost(usage, priceIn, priceOut),
		DurationMS:       time.Since(start).Milliseconds(),
		ReviewedAt:       time.Now().UTC(),
	}

	if mr != nil {
		e.MRID = mr.ID
		e.MRTitle = mr.Title
		e.MRURL = mr.WebURL
		e.AuthorUsername = mr.AuthorUsername
	}

	if uc.eventRepo != nil {
		if err := uc.eventRepo.SaveEvent(e); err != nil {
			uc.log.Error("failed to save review event", slog.String("error", err.Error()))
		}
	}

	if uc.publisher != nil {
		uc.publisher.Publish(e)
	}
}
