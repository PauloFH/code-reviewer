package usecase

import (
	"fmt"
	"log/slog"

	"github.com/paulofh/code-reviewer/internal/domain"
)

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
type ProcessMergeRequest struct {
	registry   domain.ClientRegistry
	repository domain.ReviewRepository
	reviewer   domain.AIReviewer
	log        *slog.Logger
}

func NewProcessMergeRequest(
	registry domain.ClientRegistry,
	repository domain.ReviewRepository,
	reviewer domain.AIReviewer,
	log *slog.Logger,
) *ProcessMergeRequest {
	return &ProcessMergeRequest{
		registry:   registry,
		repository: repository,
		reviewer:   reviewer,
		log:        log,
	}
}

// Execute runs the full review pipeline for a single webhook event.
// Returns nil when the event was processed (or intentionally skipped).
// Returns a non-nil error only when a recoverable step failed and the
// caller should signal an error back to the platform.
func (uc *ProcessMergeRequest) Execute(event *domain.WebhookEvent) error {
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

	// Fetch full MR to check reviewers/assignees — cheaper to do this once
	// than to try to infer assignment from the raw webhook payload.
	mr, err := client.GetMergeRequest(event.ProjectRef, event.MRIID)
	if err != nil {
		log.Error("error fetching merge request from platform", slog.String("error", err.Error()))
		return fmt.Errorf("fetching MR: %w", err)
	}

	if !mr.IsUserAssigned(username) {
		log.Info("webhook ignored: watched user is not assigned", slog.String("username", username))
		return nil
	}

	log.Info("watched user is assigned", slog.String("username", username))

	already, err := uc.repository.AlreadyReviewed(event.ClientName, event.ProjectRef, event.MRIID, username)
	if err != nil {
		log.Error("error checking idempotency", slog.String("error", err.Error()))
		return fmt.Errorf("checking idempotency: %w", err)
	}
	if already {
		log.Info("MR already reviewed, skipping duplicate", slog.String("username", username))
		return nil
	}

	log.Info("fetching diff")

	diff, err := client.GetMergeRequestDiff(event.ProjectRef, event.MRIID)
	if err != nil {
		log.Error("error fetching diff from platform", slog.String("error", err.Error()))
		return fmt.Errorf("fetching diff: %w", err)
	}

	log.Info("sending diff to AI reviewer")

	result, err := uc.reviewer.ReviewDiff(domain.ReviewInput{
		MergeRequest: mr,
		Diff:         diff,
	})
	if err != nil {
		log.Error("AI reviewer returned error", slog.String("error", err.Error()))
		return fmt.Errorf("generating review: %w", err)
	}

	if err := client.CreateMergeRequestNote(event.ProjectRef, event.MRIID, result.Body); err != nil {
		log.Error("error posting comment to platform", slog.String("error", err.Error()))
		return fmt.Errorf("posting comment: %w", err)
	}

	log.Info("review comment posted", slog.String("mr_url", mr.WebURL))

	// Best-effort: if this fails the comment was already posted.
	// The UNIQUE constraint means a manual retry will be a no-op on the DB side,
	// but the platform client would post a duplicate comment first.
	if err := uc.repository.MarkAsReviewed(event.ClientName, event.ProjectRef, event.MRIID, mr.ID, username); err != nil {
		log.Error("error persisting review record — duplicate comments possible on retry",
			slog.String("error", err.Error()))
	}

	return nil
}
